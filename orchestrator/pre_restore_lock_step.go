package orchestrator

import (
	"github.com/pkg/errors"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
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
