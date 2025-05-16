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
type ArtifactDirError customError
type DrainError customError

func NewLockError(errorMessage string) LockError {
	return LockError{errors.New(errorMessage)}
}

func NewBackupError(errorMessage string) BackupError {
	return BackupError{errors.New(errorMessage)}
}

func NewPostUnlockError(errorMessage string) UnlockError {
	return UnlockError{errors.New(errorMessage)}
}

func NewDrainError(errorMessage string) DrainError {
	return DrainError{errors.New(errorMessage)}
}

func NewCleanupError(errorMessage string) CleanupError {
	return CleanupError{errors.New(errorMessage)}
}

func NewArtifactDirError(errorMessage string) ArtifactDirError {
	return ArtifactDirError{errors.New(errorMessage)}
}

func ConvertErrors(errs []error) error {
	flattenedErrors := flattenErrors(errs)

	if len(flattenedErrors) == 0 {
		return nil
	}

	return Error(flattenedErrors)
}

func flattenErrors(errs []error) []error {
	var flattenedErrs []error

	for _, err := range errs {
		compositeError, isCompositeError := err.(Error)
		if isCompositeError {
			flattenedErrs = append(flattenedErrs, flattenErrors(compositeError)...)
		} else {
			flattenedErrs = append(flattenedErrs, err)
		}
	}

	return flattenedErrs
}

func NewError(errs ...error) Error {
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

	fmt.Fprintf(buffer, "%d error%s occurred:\n", len(err), err.getPostFix()) //nolint:errcheck
	for index, err := range err {
		fmt.Fprintf(buffer, "error %d:\n", index+1) //nolint:errcheck
		if includeStacktrace {
			fmt.Fprintf(buffer, "%+v", err) //nolint:errcheck
		} else {
			fmt.Fprintf(buffer, "%+v", err.Error()) //nolint:errcheck
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

func (err Error) ContainsUnlockOrCleanupOrArtifactDirExists() bool {
	for _, e := range err {
		switch e.(type) {
		case UnlockError:
			return true
		case CleanupError:
			return true
		case ArtifactDirError:
			return true
		case DrainError:
			return true
		default:
			continue
		}
	}

	return false
}

func (err Error) ContainsArtifactDirError() bool {
	for _, e := range err {
		_, ok := e.(ArtifactDirError)
		return ok
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
