package orchestrator

import "github.com/pkg/errors"

type PreRestoreLockStep struct{}

func NewPreRestoreLockStep() Step {
	return &PreRestoreLockStep{}
}

func (s *PreRestoreLockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PreRestoreLock()

	if err != nil {
		return errors.Wrap(err, "pre-restore-lock failed")
	}
	return nil
}
