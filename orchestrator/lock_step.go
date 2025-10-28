package orchestrator

import "github.com/cloudfoundry/bosh-backup-and-restore/executor"

type LockStep struct {
	lockOrderer LockOrderer
	executor    executor.Executor
}

func (s *LockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PreBackupLock(s.lockOrderer, s.executor)
	if err != nil {
		return NewLockError(err.Error())
	}
	return nil
}

func NewLockStep(lockOrderer LockOrderer, executor executor.Executor) Step {
	return &LockStep{lockOrderer: lockOrderer, executor: executor}
}
