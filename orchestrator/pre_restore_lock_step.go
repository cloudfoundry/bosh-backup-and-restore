package orchestrator

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/executor"
	"github.com/pkg/errors"
)

type PreRestoreLockStep struct {
	lockOrderer LockOrderer
	executor    executor.Executor
}

func NewPreRestoreLockStep(lockOrderer LockOrderer, executor executor.Executor) Step {
	return &PreRestoreLockStep{
		lockOrderer: lockOrderer,
		executor:    executor,
	}
}

func (s *PreRestoreLockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PreRestoreLock(s.lockOrderer, s.executor)

	if err != nil {
		return errors.Wrap(err, "pre-restore-lock failed")
	}
	return nil
}
