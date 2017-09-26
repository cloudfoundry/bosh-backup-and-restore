package command

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"

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

	directorName := ExtractNameFromAddress(c.Parent().String("host"))

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

func trapSigint(backup bool) {
	sigintChan := make(chan os.Signal, 1)
	signal.Notify(sigintChan, os.Interrupt)

	var sigintQuestion, stdInErrorMessage, cleanupAdvisedNotice string
	if backup {
		sigintQuestion = backupSigintQuestion
		stdInErrorMessage = backupStdinErrorMessage
		cleanupAdvisedNotice = backupCleanupAdvisedNotice
	} else {
		sigintQuestion = restoreSigintQuestion
		stdInErrorMessage = restoreStdinErrorMessage
		cleanupAdvisedNotice = restoreCleanupAdvisedNotice
	}

	go func() {
		for range sigintChan {
			stdinReader := bufio.NewReader(os.Stdin)
			factory.ApplicationLoggerStdout.Pause()
			factory.ApplicationLoggerStderr.Pause()
			fmt.Fprintln(os.Stdout, "\n"+sigintQuestion)
			input, err := stdinReader.ReadString('\n')
			if err != nil {
				fmt.Println("\n" + stdInErrorMessage)
			} else if strings.ToLower(strings.TrimSpace(input)) == "yes" {
				fmt.Println(cleanupAdvisedNotice)
				os.Exit(1)
			}
			factory.ApplicationLoggerStdout.Resume()
			factory.ApplicationLoggerStderr.Resume()
		}
	}()
}
