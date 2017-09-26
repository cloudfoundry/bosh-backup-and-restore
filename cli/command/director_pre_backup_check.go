package command

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

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
	directorName := ExtractNameFromAddress(c.Parent().String("host"))

	backuper := factory.BuildDirectorBackupChecker(
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		c.GlobalBool("debug"),
	)

	backupable, checkErr := backuper.CanBeBackedUp(directorName)

	if backupable {
		fmt.Printf("Director can be backed up.\n")
		return cli.NewExitError("", 0)
	} else {
		fmt.Printf("Director cannot be backed up.\n")
		writeStackTrace(checkErr.PrettyError(true))
		return cli.NewExitError(checkErr.Error(), 1)
	}
}
func writeStackTrace(errorWithStackTrace string) error {
	if errorWithStackTrace != "" {
		err := ioutil.WriteFile(fmt.Sprintf("bbr-%s.err.log", time.Now().UTC().Format(time.RFC3339)), []byte(errorWithStackTrace), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func ExtractNameFromAddress(address string) string {
	url, err := url.Parse(address)
	if err == nil && url.Hostname() != "" {
		address = url.Hostname()
	}
	return strings.Split(address, ":")[0]
}
