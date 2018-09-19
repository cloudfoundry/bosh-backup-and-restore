package command

import (
	"fmt"

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
	username, password, target, caCert, debug, withManifest, allDeployments := getParams(c)

	logger := factory.BuildLogger(debug)
	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	backupChecker := factory.BuildDeploymentBackupChecker(boshClient, logger, withManifest)

	var errs orchestrator.Error

	if allDeployments {
		errs = allDeploymentsBackupCheck(boshClient, backupChecker)
	} else {
		errs = backupableCheck(backupChecker, c.Parent().String("deployment"))
	}

	if errs != nil {
		return processError(errs)
	}

	return cli.NewExitError("", 0)
}

func getParams(c *cli.Context) (string, string, string, string, bool, bool, bool) {
	username := c.Parent().String("username")
	password := c.Parent().String("password")
	target := c.Parent().String("target")
	caCert := c.Parent().String("ca-cert")
	allDeployments := c.Parent().Bool("all-deployments")
	debug := c.GlobalBool("debug")
	withManifest := c.Bool("with-manifest")

	return username, password, target, caCert, debug, withManifest, allDeployments
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

func allDeploymentsBackupCheck(boshClient bosh.Client, backupChecker *orchestrator.BackupChecker) orchestrator.Error {
	var unbackupableDeploymentsErrors []error
	var unbackupableDeploymentsCount int

	allDeployments, err := boshClient.Director.Deployments()
	if err != nil {
		return orchestrator.NewError(err)
	}

	for _, deployment := range allDeployments {
		errs := backupableCheck(backupChecker, deployment.Name())
		if errs != nil {
			unbackupableDeploymentsErrors = append(unbackupableDeploymentsErrors, errs...)
			unbackupableDeploymentsCount++
		}
	}

	fmt.Printf("Found %d Deployments that can be backed up\n", len(allDeployments)-unbackupableDeploymentsCount)
	if unbackupableDeploymentsErrors != nil {
		fmt.Println("Not all deployments can be backed up")
		return unbackupableDeploymentsErrors
	}

	return nil
}
