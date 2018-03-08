package orchestrator

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"

type PostBackupUnlockStep struct {
	lockOrderer LockOrderer
	executor    executor.Executor
}

func NewPostBackupUnlockStep(lockOrderer LockOrderer, executor executor.Executor) Step {
	return &PostBackupUnlockStep{
		lockOrderer: lockOrderer,
		executor:    executor,
	}
}

func (s *PostBackupUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostBackupUnlock(s.lockOrderer, s.executor)
	if err != nil {
		return NewPostUnlockError(err.Error())
	}
	return nil
}
