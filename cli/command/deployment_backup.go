package command

import (
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/deployment"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

type DeploymentBackupCommand struct {
}

func NewDeploymentBackupCommand() DeploymentBackupCommand {
	return DeploymentBackupCommand{}
}

func (d DeploymentBackupCommand) Cli() cli.Command {
	return cli.Command{
		Name:    "backup",
		Aliases: []string{"b"},
		Usage:   "Backup a deployment",
		Action:  d.Action,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "with-manifest",
				Usage: "Download the deployment manifest",
			},
			cli.StringFlag{
				Name:  "artifact-path",
				Usage: "Specify an optional path to save the backup artifacts to",
			},
		},
	}
}

func (d DeploymentBackupCommand) Action(c *cli.Context) error {
	trapSigint(true)

	username, password, target, caCert, debug, deployment, allDeployments := getDeploymentParams(c)
	withManifest := c.Bool("with-manifest")
	artifactPath := c.String("artifact-path")

	if allDeployments {
		return backupAll(target, username, password, caCert, artifactPath, withManifest, debug)
	} else {
		return backupSingleDeployment(deployment, target, username, password, caCert, artifactPath, withManifest, debug)
	}
}

func backupAll(target, username, password, caCert, artifactPath string, withManifest, debug bool) error {
	backupAction := func(deploymentName string) orchestrator.Error {
		logger, buffer := factory.BuildBoshLoggerWithCustomBuffer(debug)
		backuper, factoryErr := factory.BuildDeploymentBackuper(target, username, password, caCert, withManifest, logger)
		if factoryErr != nil {
			return orchestrator.NewError(factoryErr)
		}

		fmt.Printf("Starting backup of %s\n", deploymentName)
		err := backuper.Backup(deploymentName, artifactPath)
		if err != nil {
			fmt.Printf("ERROR: failed to backup %s\n", deploymentName)
			fmt.Println(buffer.String())
		} else {
			fmt.Printf("Finished backup of %s\n", deploymentName)
		}

		return err
	}

	errorHandler := func(deploymentError deployment.AllDeploymentsError) error {
		if deployment.ContainsUnlockOrCleanup(deploymentError.DeploymentErrs) {
			return deploymentError.ProcessWithFooter(backupCleanupAllDeploymentsAdvisedNotice)
		}
		return deploymentError.Process()
	}

	fmt.Println("Starting backup...")

	logger, _ := factory.BuildBoshLoggerWithCustomBuffer(debug)
	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	return runForAllDeployments(backupAction,
		boshClient,
		"cannot be backed up",
		"backed up",
		errorHandler,
		deployment.NewSerialExecutor())
}

func backupSingleDeployment(deployment, target, username, password, caCert, artifactPath string, withManifest, debug bool) error {
	logger := factory.BuildBoshLogger(debug)
	backuper, err := factory.BuildDeploymentBackuper(target, username, password, caCert, withManifest, logger)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	backupErr := backuper.Backup(deployment, artifactPath)
	if backupErr.ContainsUnlockOrCleanup() {
		return processErrorWithFooter(backupErr, backupCleanupAdvisedNotice)
	} else {
		return processError(backupErr)
	}
}
