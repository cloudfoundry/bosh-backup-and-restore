package orchestrator

import (
	"errors"

	"bytes"

	"fmt"

	"github.com/mgutz/ansi"
	"github.com/urfave/cli"
)

type LockError struct {
	error
}

type BackupError struct {
	error
}

type PostBackupUnlockError struct {
	error
}

type CleanupError struct {
	error
}

func NewLockError(errorMessage string) LockError {
	return LockError{errors.New(errorMessage)}
}

func NewBackupError(errorMessage string) BackupError {
	return BackupError{errors.New(errorMessage)}
}

func NewPostBackupUnlockError(errorMessage string) PostBackupUnlockError {
	return PostBackupUnlockError{errors.New(errorMessage)}
}

func NewCleanupError(errorMessage string) CleanupError {
	return CleanupError{errors.New(errorMessage)}
}

type Error []error

func (e Error) Error() string {
	if e.IsNil() {
		return ""
	}
	var buffer *bytes.Buffer = bytes.NewBufferString("")

	errorPostfix := ""
	if len(e) > 1 {
		errorPostfix = "s"
	}
	fmt.Fprintf(buffer, "%d error%s occurred:\n", len(e), errorPostfix)
	for _, err := range e {
		fmt.Fprintf(buffer, "%+v\n", err)
	}
	return buffer.String()
}

func (e Error) IsCleanup() bool {
	if len(e) == 1 {
		_, ok := e[0].(CleanupError)
		return ok
	}

	return false
}

func (err Error) IsPostBackup() bool {
	foundPostBackupError := false

	for _, e := range err {
		switch e.(type) {
		case PostBackupUnlockError:
			foundPostBackupError = true
		case CleanupError:
			continue
		default:
			return false
		}
	}

	return foundPostBackupError
}

func (e Error) IsFatal() bool {
	return !e.IsNil() && !e.IsCleanup() && !e.IsPostBackup()
}

func (e Error) IsNil() bool {
	return len(e) == 0
}

func ProcessBackupError(errs Error) (int, string) {
	exitCode := 0

	for _, err := range errs {
		switch err.(type) {
		case LockError:
			exitCode = exitCode | 1<<2
		case PostBackupUnlockError:
			exitCode = exitCode | 1<<3
		case CleanupError:
			exitCode = exitCode | 1<<4
		default:
			exitCode = exitCode | 1
		}

	}

	return exitCode, errs.Error()
}

func ProcessRestoreError(err error) error {
	switch err := err.(type) {
	case CleanupError:
		return cli.NewExitError(ansi.Color(err.Error(), "yellow"), 2)
	case PostBackupUnlockError:
		return cli.NewExitError(ansi.Color(err.Error(), "red"), 42)
	case error:
		return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
	default:
		return err
	}
}
