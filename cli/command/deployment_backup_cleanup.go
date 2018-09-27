package command

import (
	"errors"
	"fmt"

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
	deployments, err := getAllDeployments(boshClient)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	if len(deployments) == 0 {
		return processError(orchestrator.NewError(errors.New("Failed to find any deployments")))
	}

	var cleanupErrors []deploymentError

	for _, deployment := range deployments {
		err := cleanup(cleaner, deployment.Name())
		if err != nil {
			cleanupErrors = append(cleanupErrors, deploymentError{deployment: deployment.Name(), errs: err})
		}
		fmt.Println("-------------------------")
	}

	if len(cleanupErrors) != 0 {
		errMsg := fmt.Sprintf("%d out of %d deployments could not be cleaned up:\n", len(cleanupErrors), len(deployments))
		for _, cleanupErr := range cleanupErrors {
			errMsg = errMsg + cleanupErr.deployment + "\n"
		}
		return allDeploymentsError{summary: errMsg, deploymentErrs: cleanupErrors}.Process()
	}

	fmt.Printf("All %d deployments were cleaned up.\n", len(deployments))
	return cli.NewExitError("", 0)

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
