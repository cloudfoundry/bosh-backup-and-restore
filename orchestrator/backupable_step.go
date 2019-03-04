package orchestrator

import (
	"github.com/pkg/errors"
)

type BackupableStep struct {
	lockOrderer LockOrderer
	logger      Logger
}

func NewBackupableStep(lockOrderer LockOrderer, logger Logger) Step {
	return &BackupableStep{lockOrderer: lockOrderer, logger: logger}
}

func (s *BackupableStep) Run(session *Session) error {
	s.logger.Info("bbr", "Running pre-checks for backup of %s...\n", session.DeploymentName())

	deployment := session.CurrentDeployment()
	if !deployment.IsBackupable() {
		return errors.Errorf("Deployment '%s' has no backup scripts", session.DeploymentName())
	}

	err := deployment.CheckArtifactDir()
	if err != nil {
		return NewArtifactDirError(err.Error())
	}

	if err := deployment.ValidateLockingDependencies(s.lockOrderer); err != nil {
		return err
	}
	return nil
}
