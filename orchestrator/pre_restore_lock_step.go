package orchestrator

import "github.com/pkg/errors"

type PreRestoreLockStep struct {
	lockOrderer LockOrderer
}

func NewPreRestoreLockStep(lockOrderer LockOrderer) Step {
	return &PreRestoreLockStep{lockOrderer: lockOrderer}
}

func (s *PreRestoreLockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PreRestoreLock(s.lockOrderer)

	if err != nil {
		return errors.Wrap(err, "pre-restore-lock failed")
	}
	return nil
}
