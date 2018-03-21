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
	defer s.logger.Info("bbr", "Backup created of %s on %v\n", session.DeploymentName(), time.Now())
	return s.artifactCopier.DownloadBackupFromDeployment(session.CurrentArtifact(), session.CurrentDeployment())
}
