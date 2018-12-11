package command

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/deployment"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

const artifactTimeStampFormat = "20060102T150405Z"

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
		timeStamp := time.Now().UTC().Format(artifactTimeStampFormat)

		logfilePath := filepath.Join(artifactPath, fmt.Sprintf("%s_%s.log", deploymentName, timeStamp))
		logFile, _ := os.OpenFile(logfilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, defaultLogfilePermissions)

		buffer := new(bytes.Buffer)
		multiWriter := io.MultiWriter(buffer, logFile)

		logger := factory.BuildBoshLoggerWithCustomWriter(multiWriter, debug)

		backuper, factoryErr := factory.BuildDeploymentBackuper(target, username, password, caCert, withManifest, logger, timeStamp)
		if factoryErr != nil {
			return orchestrator.NewError(factoryErr)
		}

		printlnWithTimestamp(fmt.Sprintf("Starting backup of %s, log file: %s", deploymentName, logfilePath))
		err := backuper.Backup(deploymentName, artifactPath)

		if err != nil {
			printlnWithTimestamp(fmt.Sprintf("ERROR: failed to backup %s", deploymentName))
			fmt.Println(buffer.String())
		} else {
			printlnWithTimestamp(fmt.Sprintf("Finished backup of %s", deploymentName))
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
		deployment.NewParallelExecutor())
}

func backupSingleDeployment(deployment, target, username, password, caCert, artifactPath string, withManifest, debug bool) error {
	logger := factory.BuildBoshLogger(debug)
	timeStamp := time.Now().UTC().Format(artifactTimeStampFormat)

	backuper, err := factory.BuildDeploymentBackuper(target, username, password, caCert, withManifest, logger, timeStamp)
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

func printlnWithTimestamp(str string) {
	fmt.Printf("[%s] %s\n", time.Now().UTC().Format("15:04:05"), str)
}
