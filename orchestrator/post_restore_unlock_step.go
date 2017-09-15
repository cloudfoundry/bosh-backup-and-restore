package orchestrator

import "github.com/pkg/errors"

type PostRestoreUnlockStep struct{}

func NewPostRestoreUnlockStep() Step {
	return &PostRestoreUnlockStep{}
}

func (s *PostRestoreUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostRestoreUnlock()

	if err != nil {
		return errors.Wrap(err, "post-restore-unlock failed")
	}

	return nil
}
