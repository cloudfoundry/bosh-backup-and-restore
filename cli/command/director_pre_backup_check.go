package command

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/urfave/cli"
)

type DirectorPreBackupCheckCommand struct {
}

func (checkCommand DirectorPreBackupCheckCommand) Cli() cli.Command {
	return cli.Command{
		Name:    "pre-backup-check",
		Aliases: []string{"c"},
		Usage:   "Check a BOSH Director can be backed up",
		Action:  checkCommand.Action,
	}
}

func NewDirectorPreBackupCheckCommand() DirectorPreBackupCheckCommand {
	return DirectorPreBackupCheckCommand{}
}

func (checkCommand DirectorPreBackupCheckCommand) Action(c *cli.Context) error {
	directorName := extractNameFromAddress(c.Parent().String("host"))

	backupChecker := factory.BuildDirectorBackupChecker(
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		c.GlobalBool("debug"),
	)

	err := backupChecker.Check(directorName)

	if err != nil {
		fmt.Printf("Director cannot be backed up.\n")
		return processError(err)
	}

	fmt.Printf("Director can be backed up.\n")
	return cli.NewExitError("", 0)
}
