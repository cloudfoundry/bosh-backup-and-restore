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
	username, password, target, caCert, debug, deployment, allDeployments := getDeploymentParams(c)

	logger := factory.BuildLogger(debug)
	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	backupChecker := factory.BuildDeploymentBackupChecker(boshClient, logger, false)

	if allDeployments {
		errs := allDeploymentsBackupCheck(boshClient, backupChecker)
		if errs != nil {
			return errs
		}
	} else {
		errs := backupableCheck(backupChecker, deployment)
		if errs != nil {
			return processError(errs)
		}
	}

	return cli.NewExitError("", 0)
}

func backupableCheck(backupChecker *orchestrator.BackupChecker, deployment string) orchestrator.Error {
	err := backupChecker.Check(deployment)
	if err != nil {
		fmt.Printf("Deployment '%s' cannot be backed up.\n", deployment)
		fmt.Println(err.Error())
		return err
	}

	fmt.Printf("Deployment '%s' can be backed up.\n", deployment)
	return nil
}

func allDeploymentsBackupCheck(boshClient bosh.Client, backupChecker *orchestrator.BackupChecker) error {
	backupCheckerAction := func(deploymentName string) orchestrator.Error {
		return backupableCheck(backupChecker, deploymentName)
	}

	errorHandler := func(deploymentError allDeploymentsError) error {
		return deploymentError.Process()
	}

	return runForAllDeployments(backupCheckerAction,
		boshClient,
		"cannot be backed up",
		"can be backed up",
		errorHandler)
}
