package orchestrator

import "github.com/hashicorp/go-multierror"

type CleanupError struct {
	error
}

type PostBackupUnlockError struct {
	error
}

type Error []error

func (e Error) Error() string {
	return multierror.ListFormatFunc(e)
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
