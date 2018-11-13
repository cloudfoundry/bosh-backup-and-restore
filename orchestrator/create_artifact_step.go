package orchestrator

import (
	"fmt"
	"time"
)

type CreateArtifactStep struct {
	logger            Logger
	backupManager     BackupManager
	deploymentManager DeploymentManager
	nowFunc           func() time.Time
	timeStamp         string
}

func (s *CreateArtifactStep) Run(session *Session) error {
	s.logger.Info("bbr", "Starting backup of %s...\n", session.DeploymentName())

	directoryName := fmt.Sprintf("%s_%s", session.DeploymentName(), s.timeStamp)
	artifact, err := s.backupManager.Create(session.CurrentArtifactPath(), directoryName, s.logger)
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

func NewCreateArtifactStep(logger Logger, backupManager BackupManager, deploymentManager DeploymentManager, nowFunc func() time.Time, timeStamp string) Step {
	return &CreateArtifactStep{logger: logger, backupManager: backupManager, deploymentManager: deploymentManager, nowFunc: nowFunc, timeStamp: timeStamp}
}
