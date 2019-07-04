package command

import (
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
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
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "artifact-path, a",
				Usage: "Specify an optional path to save the backup artifacts to",
			},
		},
	}

}

func (checkCommand DirectorBackupCommand) Action(c *cli.Context) error {
	trapSigint(true)

	directorName := extractNameFromAddress(c.Parent().String("host"))
	timeStamp := time.Now().UTC().Format(artifactTimeStampFormat)

	backuper := factory.BuildDirectorBackuper(
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		c.App.Version,
		c.GlobalBool("debug"),
		timeStamp)

	backupErr := backuper.Backup(directorName, c.String("artifact-path"))

	if backupErr.ContainsUnlockOrCleanupOrArtifactDirExists() {
		return processErrorWithFooter(backupErr, backupCleanupAdvisedNotice)
	} else {
		return processError(backupErr)
	}
}
