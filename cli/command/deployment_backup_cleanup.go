package command

import (
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/deployment"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

type DeploymentBackupCleanupCommand struct {
}

func NewDeploymentBackupCleanupCommand() DeploymentBackupCleanupCommand {
	return DeploymentBackupCleanupCommand{}
}

func (d DeploymentBackupCleanupCommand) Cli() cli.Command {
	return cli.Command{
		Name:   "backup-cleanup",
		Usage:  "Cleanup a deployment after a backup was interrupted",
		Action: d.Action,
	}
}

func (d DeploymentBackupCleanupCommand) Action(c *cli.Context) error {
	trapSigint(true)

	username, password, target, caCert, debug, deployment, allDeployments := getDeploymentParams(c)
	logger := factory.BuildLogger(debug)
	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)

	cleaner, err := factory.BuildDeploymentBackupCleanuper(
		c.Bool("with-manifest"),
		boshClient,
		logger,
	)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	if !allDeployments {
		cleanupErr := cleaner.Cleanup(deployment)
		return processError(cleanupErr)
	}

	return cleanupAllDeployments(cleaner, boshClient)

}
func cleanupAllDeployments(cleaner *orchestrator.BackupCleaner, boshClient bosh.Client) error {
	cleanupAction := func(deploymentName string) orchestrator.Error {
		return cleanup(cleaner, deploymentName)
	}

	errorHandler := func(deploymentError deployment.AllDeploymentsError) error {
		return deploymentError.Process()
	}
	return runForAllDeployments(
		cleanupAction,
		boshClient,
		"could not be cleaned up",
		"were cleaned up",
		errorHandler,
		deployment.NewSerialExecutor())
}

func cleanup(cleaner *orchestrator.BackupCleaner, deployment string) orchestrator.Error {
	err := cleaner.Cleanup(deployment)
	if err != nil {
		fmt.Printf("Failed to cleanup deployment '%s'\n", deployment)
		return err
	}
	fmt.Printf("Cleaned up deployment '%s'\n", deployment)
	return nil
}
