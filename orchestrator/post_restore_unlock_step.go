package orchestrator

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"

type PostRestoreUnlockStep struct {
	lockOrderer LockOrderer
	executor    executor.Executor
}

func NewPostRestoreUnlockStep(lockOrderer LockOrderer, executor executor.Executor) Step {
	return &PostRestoreUnlockStep{
		lockOrderer: lockOrderer,
		executor:    executor,
	}
}

func (s *PostRestoreUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostRestoreUnlock(s.lockOrderer, s.executor)

	if err != nil {
		return NewPostUnlockError(err.Error())
	}

	return nil
}
