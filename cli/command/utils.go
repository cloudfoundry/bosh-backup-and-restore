package command

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"io/ioutil"
	"time"

	"net/url"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

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

func processError(err orchestrator.Error) error {
	errorCode, errorMessage, errorWithStackTrace := orchestrator.ProcessError(err)
	if err := writeStackTrace(errorWithStackTrace); err != nil {
		return errors.Wrap(err, err.Error())
	}

	return cli.NewExitError(errorMessage, errorCode)
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

func extractNameFromAddress(address string) string {
	url, err := url.Parse(address)
	if err == nil && url.Hostname() != "" {
		address = url.Hostname()
	}
	return strings.Split(address, ":")[0]
}

func redCliError(err error) *cli.ExitError {
	return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
}
