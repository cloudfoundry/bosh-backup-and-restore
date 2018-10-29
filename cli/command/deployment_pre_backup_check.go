package command

import (
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/deployment"
	"github.com/cloudfoundry/bosh-utils/logger"

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
	var logger logger.Logger
	if allDeployments {
		logger, _ = factory.BuildBoshLoggerWithCustomBuffer(debug)
	} else {
		logger = factory.BuildBoshLogger(debug)
	}
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
			if errs.ContainsArtifactDirError() {
				return processErrorWithFooter(errs, backupCleanupAdvisedNotice)
			}
			return processError(errs)
		}
	}

	return cli.NewExitError("", 0)
}

func backupableCheck(backupChecker *orchestrator.BackupChecker, deploymentName string) orchestrator.Error {
	err := backupChecker.Check(deploymentName)

	if err != nil {
		fmt.Printf("Deployment '%s' cannot be backed up.\n", deploymentName)
		fmt.Println(deployment.IndentBlock(err.Error()))
		return err
	}

	fmt.Printf("Deployment '%s' can be backed up.\n", deploymentName)
	return nil
}

func allDeploymentsBackupCheck(boshClient bosh.Client, backupChecker *orchestrator.BackupChecker) error {
	backupCheckerAction := func(deploymentName string) orchestrator.Error {
		return backupableCheck(backupChecker, deploymentName)
	}

	errorHandler := func(deploymentError deployment.AllDeploymentsError) error {
		if deployment.ContainsArtifactDir(deploymentError.DeploymentErrs) {
			return deploymentError.ProcessWithFooter(backupCleanupAllDeploymentsAdvisedNotice)
		}
		return deploymentError.Process()
	}

	return runForAllDeployments(backupCheckerAction,
		boshClient,
		"cannot be backed up",
		"can be backed up",
		errorHandler,
		deployment.NewParallelExecutor(),
	)
}
