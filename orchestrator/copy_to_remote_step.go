package orchestrator

import "github.com/pkg/errors"

type CopyToRemoteStep struct{}

func NewCopyToRemoteStep() Step {
	return &CopyToRemoteStep{}
}

func (s *CopyToRemoteStep) Run(session *Session) error {
	if err := session.CurrentDeployment().CopyLocalBackupToRemote(session.CurrentArtifact()); err != nil {
		return errors.Errorf("Unable to send backup to remote machine. Got error: %s", err)
	}
	return nil
}
