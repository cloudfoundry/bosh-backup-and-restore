package command

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cli/director"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

type DeploymentPreBackupCheck struct{}

func NewDeploymentPreBackupCheckCommand() DeploymentPreBackupCheck {
	return DeploymentPreBackupCheck{}
}
func (d DeploymentPreBackupCheck) Cli() cli.Command {
	return cli.Command{
		Name:    "pre-backup-check",
		Aliases: []string{"c"},
		Usage:   "Check a deployment can be backed up",
		Action:  d.Action,
		Flags:   []cli.Flag{},
	}
}

func (d DeploymentPreBackupCheck) Action(c *cli.Context) error {
	username, password, target, caCert, debug, deployment, allDeployments := getDeploymentParams(c)

	logger := factory.BuildLogger(debug)
	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	backupChecker := factory.BuildDeploymentBackupChecker(boshClient, logger, false)

	var errs orchestrator.Error

	if allDeployments {
		errs = allDeploymentsBackupCheck(boshClient, backupChecker)
	} else {
		errs = backupableCheck(backupChecker, deployment)
	}

	if errs != nil {
		return processError(errs)
	}

	return cli.NewExitError("", 0)
}

func getDeploymentParams(c *cli.Context) (string, string, string, string, bool, string, bool) {
	username := c.Parent().String("username")
	password := c.Parent().String("password")
	target := c.Parent().String("target")
	caCert := c.Parent().String("ca-cert")
	debug := c.GlobalBool("debug")
	deployment := c.Parent().String("deployment")
	allDeployments := c.Parent().Bool("all-deployments")

	return username, password, target, caCert, debug, deployment, allDeployments
}

func backupableCheck(backupChecker *orchestrator.BackupChecker, deployment string) orchestrator.Error {
	backupable, checkErr := backupChecker.CanBeBackedUp(deployment)
	if backupable {
		fmt.Printf("Deployment '%s' can be backed up.\n", deployment)
		return nil
	} else {
		fmt.Printf("Deployment '%s' cannot be backed up.\n", deployment)
		return checkErr
	}
}

func getAllDeployments(boshClient bosh.Client) ([]director.Deployment, error) {
	allDeployments, err := boshClient.Director.Deployments()
	if err != nil {
		return nil, orchestrator.NewError(err)
	}

	fmt.Printf("Found %d deployments:\n", len(allDeployments))
	for _, deployment := range allDeployments {
		fmt.Printf("%s\n", deployment.Name())
	}
	fmt.Println("-------------------------")

	return allDeployments, nil
}

func allDeploymentsBackupCheck(boshClient bosh.Client, backupChecker *orchestrator.BackupChecker) orchestrator.Error {
	var unbackupableDeploymentsErrors []error
	var unbackupableDeploymentNames []string

	allDeployments, err := getAllDeployments(boshClient)
	if err != nil {
		return orchestrator.NewError(err)
	}

	for _, deployment := range allDeployments {
		errs := backupableCheck(backupChecker, deployment.Name())
		if errs != nil {
			unbackupableDeploymentsErrors = append(unbackupableDeploymentsErrors, errs...)
			unbackupableDeploymentNames = append(unbackupableDeploymentNames, deployment.Name())
		}
		fmt.Println("-------------------------")
	}

	if unbackupableDeploymentsErrors != nil {
		fmt.Printf("%d out of %d deployments cannot be backed up:\n", len(unbackupableDeploymentNames), len(allDeployments))
		for _, deploymentName := range unbackupableDeploymentNames {
			fmt.Println(deploymentName)
		}
		fmt.Println("")
		return unbackupableDeploymentsErrors
	}

	fmt.Printf("All %d deployments can be backed up.\n", len(allDeployments))
	return nil
}
