package command

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
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
		Flags: []cli.Flag{cli.BoolFlag{
			Name:  "with-manifest",
			Usage: "Download the deployment manifest",
		}},
	}
}

func (d DeploymentBackupCommand) Action(c *cli.Context) error {
	trapSigint(true)

	backuper, err := factory.BuildDeploymentBackuper(c.Parent().String("target"),
		c.Parent().String("username"),
		c.Parent().String("password"),
		c.Parent().String("ca-cert"),
		c.Bool("with-manifest"),
		c.GlobalBool("debug"),
	)

	if err != nil {
		return redCliError(err)
	}

	deployment := c.Parent().String("deployment")
	backupErr := backuper.Backup(deployment)

	errorCode, errorMessage, errorWithStackTrace := orchestrator.ProcessError(backupErr)
	if err := writeStackTrace(errorWithStackTrace); err != nil {
		return errors.Wrap(backupErr, err.Error())
	}

	if backupErr.ContainsUnlockOrCleanup() {
		errorMessage = errorMessage + "\n" + backupCleanupAdvisedNotice
	}

	return cli.NewExitError(errorMessage, errorCode)
}
