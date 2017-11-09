package command

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/cli/flags"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

type DeploymentRestoreCommand struct {
}

func NewDeploymentRestoreCommand() DeploymentRestoreCommand {
	return DeploymentRestoreCommand{}
}

func (d DeploymentRestoreCommand) Cli() cli.Command {
	return cli.Command{
		Name:    "restore",
		Aliases: []string{"r"},
		Usage:   "Restore a deployment from backup",
		Action:  d.Action,
		Flags: []cli.Flag{cli.StringFlag{
			Name:  "artifact-path",
			Usage: "Path to the artifact to restore",
		}},
	}
}

func (d DeploymentRestoreCommand) Action(c *cli.Context) error {
	trapSigint(false)

	if err := flags.Validate([]string{"artifact-path"}, c); err != nil {
		return err
	}

	deployment := c.Parent().String("deployment")
	artifactPath := c.String("artifact-path")

	restorer, err := factory.BuildDeploymentRestorer(c.Parent().String("target"),
		c.Parent().String("username"),
		c.Parent().String("password"),
		c.Parent().String("ca-cert"),
		c.GlobalBool("debug"))

	if err != nil {
		return processError(orchestrator.NewError(err))
	}

	restoreErr := restorer.Restore(deployment, artifactPath)
	return processError(restoreErr)
}
