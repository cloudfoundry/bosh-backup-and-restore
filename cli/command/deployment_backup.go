package command

import (
	"bytes"
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/deployment"
	"github.com/cloudfoundry/bosh-utils/logger"

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

	buffer := new(bytes.Buffer)
	var logger logger.Logger
	if allDeployments {
		logger = factory.BuildBoshLoggerWithCustomBuffer(debug, buffer)
	} else {
		logger = factory.BuildBoshLogger(debug)
	}

	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	if !allDeployments {
		backuper := factory.BuildDeploymentBackuper(withManifest, boshClient, logger)
		backupErr := backuper.Backup(deployment, artifactPath)

		if backupErr.ContainsUnlockOrCleanup() {
			return processErrorWithFooter(backupErr, backupCleanupAdvisedNotice)
		} else {
			return processError(backupErr)
		}
	}

	return backupAll(target, username, password, caCert, artifactPath, boshClient, withManifest, debug)
}

func backupAll(target, username, password, caCert, artifactPath string, boshClient bosh.Client, withManifest, debug bool) error {
	backupAction := func(deploymentName string) orchestrator.Error {
		buffer := new(bytes.Buffer)
		logger := factory.BuildBoshLoggerWithCustomBuffer(debug, buffer)
		boshClient, _ := factory.BuildBoshClient(target, username, password, caCert, logger)
		backuper := factory.BuildDeploymentBackuper(withManifest, boshClient, logger)

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

	return runForAllDeployments(backupAction,
		boshClient,
		"cannot be backed up",
		"backed up",
		errorHandler,
		deployment.NewSerialExecutor())
}
