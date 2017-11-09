package command

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type DirectorBackupCommand struct {
}

func NewDirectorBackupCommand() DirectorBackupCommand {
	return DirectorBackupCommand{}
}

func (checkCommand DirectorBackupCommand) Cli() cli.Command {
	return cli.Command{
		Name:    "backup",
		Aliases: []string{"b"},
		Usage:   "Backup a BOSH Director",
		Action:  checkCommand.Action,
	}

}

func (checkCommand DirectorBackupCommand) Action(c *cli.Context) error {
	trapSigint(true)

	directorName := extractNameFromAddress(c.Parent().String("host"))

	backuper := factory.BuildDirectorBackuper(
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		c.GlobalBool("debug"))

	backupErr := backuper.Backup(directorName)

	errorCode, errorMessage, errorWithStackTrace := orchestrator.ProcessError(backupErr)
	if err := writeStackTrace(errorWithStackTrace); err != nil {
		return errors.Wrap(backupErr, err.Error())
	}

	if backupErr.ContainsUnlockOrCleanup() {
		errorMessage = errorMessage + "\n" + backupCleanupAdvisedNotice
	}

	return cli.NewExitError(errorMessage, errorCode)
}
