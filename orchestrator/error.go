package orchestrator

import (
	"bytes"

	"fmt"

	"github.com/pkg/errors"
)

type customError struct {
	error
}

type LockError customError
type BackupError customError
type UnlockError customError
type CleanupError customError

func NewLockError(errorMessage string) LockError {
	return LockError{errors.New(errorMessage)}
}

func NewBackupError(errorMessage string) BackupError {
	return BackupError{errors.New(errorMessage)}
}

func NewPostUnlockError(errorMessage string) UnlockError {
	return UnlockError{errors.New(errorMessage)}
}

func NewCleanupError(errorMessage string) CleanupError {
	return CleanupError{errors.New(errorMessage)}
}

func ConvertErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return Error(errs)
}

func NewError(errs ...error) Error {
	if len(errs) == 0 {
		return nil
	}
	return Error(errs)
}

type Error []error

func (err Error) Error() string {
	return err.PrettyError(false)
}

func (err Error) PrettyError(includeStacktrace bool) string {
	if err.IsNil() {
		return ""
	}
	var buffer = bytes.NewBufferString("")

	fmt.Fprintf(buffer, "%d error%s occurred:\n", len(err), err.getPostFix())
	for index, err := range err {
		fmt.Fprintf(buffer, "error %d:\n", index+1)
		if includeStacktrace {
			fmt.Fprintf(buffer, "%+v\n", err)
		} else {
			fmt.Fprintf(buffer, "%+v\n", err.Error())
		}
	}
	return buffer.String()
}

func (err Error) getPostFix() string {
	errorPostfix := ""
	if len(err) > 1 {
		errorPostfix = "s"
	}
	return errorPostfix
}

func (err Error) ContainsUnlockOrCleanup() bool {
	for _, e := range err {
		switch e.(type) {
		case UnlockError:
			return true
		case CleanupError:
			return true
		default:
			continue
		}
	}

	return false
}

func (err Error) IsCleanup() bool {
	if len(err) == 1 {
		_, ok := err[0].(CleanupError)
		return ok
	}

	return false
}

func (err Error) IsPostBackup() bool {
	foundPostBackupError := false

	for _, e := range err {
		switch e.(type) {
		case UnlockError:
			foundPostBackupError = true
		case CleanupError:
			continue
		default:
			return false
		}
	}

	return foundPostBackupError
}

func (err Error) IsFatal() bool {
	return !err.IsNil() && !err.IsCleanup() && !err.IsPostBackup()
}

func (err Error) IsNil() bool {
	return len(err) == 0
}

func BuildExitCode(errs Error) int {
	exitCode := 0

	for _, err := range errs {
		switch err.(type) {
		case LockError:
			exitCode = exitCode | 1<<2
		case UnlockError:
			exitCode = exitCode | 1<<3
		case CleanupError:
			exitCode = exitCode | 1<<4
		default:
			exitCode = exitCode | 1
		}
	}

	return exitCode
}
