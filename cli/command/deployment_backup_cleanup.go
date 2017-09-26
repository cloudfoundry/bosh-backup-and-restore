package command

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
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

	cleaner, err := factory.BuildDeploymentBackupCleanuper(
		c.Parent().String("target"),
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
	cleanupErr := cleaner.Cleanup(deployment)

	return processError(cleanupErr)
}
