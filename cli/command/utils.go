package command

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"time"

	"net/url"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/factory"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"
)

const defaultLogfilePermissions = 0644

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
			fmt.Fprintln(os.Stdout, "\n"+sigintQuestion) //nolint:errcheck
			input, err := stdinReader.ReadString('\n')
			if err != nil {
				fmt.Println("\n" + stdInErrorMessage)
			} else if strings.ToLower(strings.TrimSpace(input)) == "yes" {
				fmt.Println(cleanupAdvisedNotice)
				os.Exit(1)
			}
			factory.ApplicationLoggerStdout.Resume() //nolint:errcheck
			factory.ApplicationLoggerStderr.Resume() //nolint:errcheck
		}
	}()
}

func processError(err orchestrator.Error) error {
	return processErrorWithFooter(err, "")
}

func processErrorWithFooter(err orchestrator.Error, footer string) error {
	errorCode := orchestrator.BuildExitCode(err)
	errorMessage := err.Error()
	errorWithStackTrace := err.PrettyError(true)

	writeErr := writeStackTrace(errorWithStackTrace)
	if writeErr != nil {
		errorMessage = errorWithStackTrace
	}

	errorMessage = errorMessage + "\n" + footer

	return cli.NewExitError(errorMessage, errorCode)
}

func writeStackTrace(errorWithStackTrace string) error {
	if errorWithStackTrace != "" {
		err := os.WriteFile(fmt.Sprintf("bbr-%s.err.log", time.Now().UTC().Format(time.RFC3339)), []byte(errorWithStackTrace), defaultLogfilePermissions)
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
