package orchestrator

import (
	"time"
)

type DrainStep struct {
	logger         Logger
	artifactCopier ArtifactCopier
}

func NewDrainStep(logger Logger, artifactCopier ArtifactCopier) Step {
	return &DrainStep{
		logger:         logger,
		artifactCopier: artifactCopier,
	}
}

func (s *DrainStep) Run(session *Session) error {
	err := s.artifactCopier.DownloadBackupFromDeployment(session.CurrentArtifact(), session.CurrentDeployment())
	if err != nil {
		s.logger.Info("bbr", "Failed to create backup of %s on %v, failed during drain step\n", session.DeploymentName(), time.Now())
		return NewDrainError(err.Error())
	}
	s.logger.Info("bbr", "Backup created of %s on %v\n", session.DeploymentName(), time.Now())
	return nil
}
