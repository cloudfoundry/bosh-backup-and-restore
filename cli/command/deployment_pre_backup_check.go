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
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "all-deployments",
				Usage: "To check if all deployments are backupable",
			},
		},
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

	if allDeployments {
		err = allDeploymentsBackupCheck(boshClient, backupChecker)
	} else {
		err = backupableCheck(backupChecker, c.Parent().String("deployment"))
	}

	if err != nil {
		return err
	}

	return cli.NewExitError("", 0)
}

func getParams(c *cli.Context) (string, string, string, string, bool, bool, bool) {
	username := c.Parent().String("username")
	password := c.Parent().String("password")
	target := c.Parent().String("target")
	caCert := c.Parent().String("ca-cert")
	debug := c.GlobalBool("debug")
	withManifest := c.Bool("with-manifest")
	allDeployments := c.Bool("all-deployments")

	return username, password, target, caCert, debug, withManifest, allDeployments
}

func backupableCheck(backupChecker *orchestrator.BackupChecker, deployment string) error {
	backupable, checkErr := backupChecker.CanBeBackedUp(deployment)
	if backupable {
		fmt.Printf("Deployment '%s' can be backed up.\n", deployment)
		return nil
	} else {
		fmt.Printf("Deployment '%s' cannot be backed up.\n", deployment)
		return processError(checkErr)
	}
}

func allDeploymentsBackupCheck(boshClient bosh.Client, backupChecker *orchestrator.BackupChecker) error {
	var backupableDeploymentsErrors []error

	allDeployments, err := boshClient.Director.Deployments()
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	for _, deployment := range allDeployments {
		err := backupableCheck(backupChecker, deployment.Name())
		if err != nil {
			backupableDeploymentsErrors = append(backupableDeploymentsErrors, err)
		}
	}

	fmt.Printf("Found %d Deployments that can be backed up\n", len(allDeployments)-len(backupableDeploymentsErrors))
	if backupableDeploymentsErrors != nil {
		fmt.Println("Not all deployments can be backed up")
		return processError(orchestrator.NewError(backupableDeploymentsErrors...))
	}

	return nil
}
