package orchestrator

import "github.com/cloudfoundry/bosh-backup-and-restore/executor"

type PostBackupUnlockStep struct {
	afterSuccessfulBackup bool
	lockOrderer           LockOrderer
	executor              executor.Executor
}

func NewPostBackupUnlockStep(afterSuccessfulBackup bool, lockOrderer LockOrderer, executor executor.Executor) Step {
	return &PostBackupUnlockStep{
		afterSuccessfulBackup: afterSuccessfulBackup,
		lockOrderer:           lockOrderer,
		executor:              executor,
	}
}

func (s *PostBackupUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostBackupUnlock(s.afterSuccessfulBackup, s.lockOrderer, s.executor)
	if err != nil {
		return NewPostUnlockError(err.Error())
	}
	return nil
}
