package command

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/urfave/cli"
)

type DirectorRestoreCleanupCommand struct {
}

func NewDirectorRestoreCleanupCommand() DirectorRestoreCleanupCommand {
	return DirectorRestoreCleanupCommand{}
}
func (d DirectorRestoreCleanupCommand) Cli() cli.Command {
	return cli.Command{
		Name:   "restore-cleanup",
		Usage:  "Cleanup a director after a restore was interrupted",
		Action: d.Action,
	}
}

func (d DirectorRestoreCleanupCommand) Action(c *cli.Context) error {
	trapSigint(true)

	directorName := extractNameFromAddress(c.Parent().String("host"))

	rateLimiter, err := getConnectionRateLimiter(c)

	if err != nil {
		return err
	}

	cleaner := factory.BuildDirectorRestoreCleaner(
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		c.App.Version,
		c.GlobalBool("debug"),
		rateLimiter,
	)

	cleanupErr := cleaner.Cleanup(directorName)

	return processError(cleanupErr)
}
