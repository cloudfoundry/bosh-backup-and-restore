package command

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/deployment"

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

	if !allDeployments {
		logger := factory.BuildBoshLogger(debug)

		cleaner, err := factory.BuildDeploymentBackupCleanuper(
			target,
			username,
			password,
			caCert,
			logger,
		)
		if err != nil {
			return processError(orchestrator.NewError(err))
		}

		cleanupErr := cleaner.Cleanup(deployment)
		return processError(cleanupErr)
	}

	return cleanupAllDeployments(target, username, password, caCert, debug)

}
func cleanupAllDeployments(target, username, password, caCert string, debug bool) error {
	cleanupAction := func(deploymentName string) orchestrator.Error {
		logger, buffer := factory.BuildBoshLoggerWithCustomBuffer(debug)
		timeStamp := time.Now().UTC().Format(artifactTimeStampFormat)
		cleaner, factoryError := factory.BuildDeploymentBackupCleanuper(
			target,
			username,
			password,
			caCert,
			logger,
		)

		if factoryError != nil {
			return orchestrator.NewError(factoryError)
		}

		logFilePath := fmt.Sprintf("%s_%s.log", deploymentName, timeStamp)

		printlnWithTimestamp(fmt.Sprintf("Starting cleanup of %s, log file: %s", deploymentName, logFilePath))
		err := cleanup(cleaner, deploymentName)

		ioutil.WriteFile(logFilePath, buffer.Bytes(), defaultLogfilePermissions)

		if err != nil {
			printlnWithTimestamp(fmt.Sprintf("ERROR: failed cleanup of %s", deploymentName))
			fmt.Println(buffer.String())
		} else {
			printlnWithTimestamp(fmt.Sprintf("Finished cleanup of %s", deploymentName))
		}

		return err
	}

	errorHandler := func(deploymentError deployment.AllDeploymentsError) error {
		return deploymentError.Process()
	}

	logger, _ := factory.BuildBoshLoggerWithCustomBuffer(debug)

	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)
	if err != nil {
		return err
	}

	fmt.Println("Starting cleanup...")

	return runForAllDeployments(
		cleanupAction,
		boshClient,
		"could not be cleaned up",
		"cleaned up",
		errorHandler,
		deployment.NewParallelExecutor())
}

func cleanup(cleaner *orchestrator.BackupCleaner, deployment string) orchestrator.Error {
	err := cleaner.Cleanup(deployment)
	if err != nil {
		fmt.Printf("Failed to cleanup deployment '%s'\n", deployment)
		return err
	}
	return nil
}
