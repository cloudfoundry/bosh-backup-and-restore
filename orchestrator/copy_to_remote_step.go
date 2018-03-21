package orchestrator

import (
	"github.com/pkg/errors"
)

type CopyToRemoteStep struct {
	artifactCopier ArtifactCopier
}

func NewCopyToRemoteStep(artifactCopier ArtifactCopier) Step {
	return &CopyToRemoteStep{
		artifactCopier: artifactCopier,
	}
}

func (s *CopyToRemoteStep) Run(session *Session) error {
	err := s.artifactCopier.UploadBackupToDeployment(session.CurrentArtifact(), session.CurrentDeployment())
	if err != nil {
		return errors.Errorf("Unable to send backup to remote machine. Got error: %s", err)
	}
	return nil
}
