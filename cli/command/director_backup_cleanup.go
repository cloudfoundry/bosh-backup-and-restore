package command

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/factory"
	"github.com/urfave/cli"
)

type DirectorBackupCleanupCommand struct {
}

func NewDirectorBackupCleanupCommand() DirectorBackupCleanupCommand {
	return DirectorBackupCleanupCommand{}
}
func (d DirectorBackupCleanupCommand) Cli() cli.Command {
	return cli.Command{
		Name:   "backup-cleanup",
		Usage:  "Cleanup a director after a backup was interrupted",
		Action: d.Action,
	}
}

func (d DirectorBackupCleanupCommand) Action(c *cli.Context) error {
	trapSigint(true)

	directorName := extractNameFromAddress(c.Parent().String("host"))

	cleaner := factory.BuildDirectorBackupCleaner(c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		c.App.Version,
		c.GlobalBool("debug"),
	)

	cleanupErr := cleaner.Cleanup(directorName)

	return processError(cleanupErr)
}
