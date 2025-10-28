package command

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

type DeploymentRestoreCleanupCommand struct {
}

func NewDeploymentRestoreCleanupCommand() DeploymentRestoreCleanupCommand {
	return DeploymentRestoreCleanupCommand{}
}

func (d DeploymentRestoreCleanupCommand) Cli() cli.Command {
	return cli.Command{
		Name:   "restore-cleanup",
		Usage:  "Cleanup a deployment after a restore was interrupted",
		Action: d.Action,
	}
}

func (d DeploymentRestoreCleanupCommand) Action(c *cli.Context) error {
	trapSigint(true)

	cleaner, err := factory.BuildDeploymentRestoreCleanuper(c.Parent().String("target"),
		c.Parent().String("username"),
		c.Parent().String("password"),
		c.Parent().String("ca-cert"),
		c.App.Version,
		c.Bool("with-manifest"),
		c.GlobalBool("debug"))

	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	deployment := c.Parent().String("deployment")
	cleanupErr := cleaner.Cleanup(deployment)

	return processError(cleanupErr)
}
