package orchestrator

import "time"

type CreateArtifactStep struct {
	logger            Logger
	backupManager     BackupManager
	deploymentManager DeploymentManager
	nowFunc           func() time.Time
}

func (s *CreateArtifactStep) Run(session *Session) error {
	s.logger.Info("bbr", "Starting backup of %s...\n", session.DeploymentName())
	artifact, err := s.backupManager.Create(session.DeploymentName(), s.logger, time.Now)
	if err != nil {
		return err
	}
	artifact.CreateMetadataFileWithStartTime(s.nowFunc())
	session.SetCurrentArtifact(artifact)

	err = s.deploymentManager.SaveManifest(session.DeploymentName(), artifact)
	if err != nil {
		return err
	}
	return nil
}

func NewCreateArtifactStep(logger Logger, backupManager BackupManager, deploymentManager DeploymentManager, nowFunc func() time.Time) Step {
	return &CreateArtifactStep{logger: logger, backupManager: backupManager, deploymentManager: deploymentManager, nowFunc: nowFunc}
}
