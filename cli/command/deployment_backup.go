package command

import (
	"fmt"

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

	backuper, err := factory.BuildDeploymentBackuper(target,
		username,
		password,
		caCert,
		withManifest,
		debug,
	)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	if !allDeployments {
		backupErr := backuper.Backup(deployment, artifactPath)

		if backupErr.ContainsUnlockOrCleanup() {
			return processErrorWithFooter(backupErr, backupCleanupAdvisedNotice)
		} else {
			return processError(backupErr)
		}
	}

	return backupAll(backuper, target, username, password, caCert, artifactPath, debug)
}

func backupAll(backuper *orchestrator.Backuper, target, username, password, caCert, artifactPath string, debug bool) error {
	logger := factory.BuildLogger(debug)
	boshClient, err := factory.BuildBoshClient(target, username, password, caCert, logger)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	deployments, err := getAllDeployments(boshClient)
	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	var errors orchestrator.Error
	for _, deployment := range deployments {
		err := backuper.Backup(deployment.Name(), artifactPath)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if errors != nil {
		if errors.ContainsUnlockOrCleanup() {
			return processErrorWithFooter(errors, backupCleanupAdvisedNotice)
		}
		return processError(errors)
	}

	fmt.Printf("All %d deployments backed up.\n", len(deployments))

	return cli.NewExitError("", 0)
}
