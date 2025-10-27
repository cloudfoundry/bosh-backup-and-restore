package command

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/cli/flags"
	"github.com/cloudfoundry/bosh-backup-and-restore/factory"
	"github.com/urfave/cli"
)

type DirectorRestoreCommand struct {
}

func NewDirectorRestoreCommand() DirectorRestoreCommand {
	return DirectorRestoreCommand{}
}

func (cmd DirectorRestoreCommand) Cli() cli.Command {
	return cli.Command{
		Name:    "restore",
		Aliases: []string{"r"},
		Usage:   "Restore a deployment from backup",
		Action:  cmd.Action,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "artifact-path, a",
				Usage: "Path to the artifact to restore",
			},
		},
	}
}

func (cmd DirectorRestoreCommand) Action(c *cli.Context) error {
	trapSigint(false)

	if err := flags.Validate([]string{"artifact-path"}, c); err != nil {
		return err
	}

	directorName := extractNameFromAddress(c.Parent().String("host"))
	artifactPath := c.String("artifact-path")

	restorer := factory.BuildDirectorRestorer(
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		c.App.Version,
		c.GlobalBool("debug"),
	)

	restoreErr := restorer.Restore(directorName, artifactPath)
	return processError(restoreErr)
}
