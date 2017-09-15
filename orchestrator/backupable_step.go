package orchestrator

import "github.com/pkg/errors"

type BackupableStep struct {
	lockOrderer LockOrderer
}

func NewBackupableStep(lockOrderer LockOrderer) Step {
	return &BackupableStep{lockOrderer: lockOrderer}
}

func (s *BackupableStep) Run(session *Session) error {
	deployment := session.CurrentDeployment()
	if !deployment.IsBackupable() {
		return errors.Errorf("Deployment '%s' has no backup scripts", session.DeploymentName())
	}

	err := deployment.CheckArtifactDir()
	if err != nil {
		return err
	}

	if !deployment.HasUniqueCustomArtifactNames() {
		return errors.Errorf("Multiple jobs in deployment '%s' specified the same backup name", session.DeploymentName())
	}

	if err := deployment.CustomArtifactNamesMatch(); err != nil {
		return err
	}

	if err := deployment.ValidateLockingDependencies(s.lockOrderer); err != nil {
		return err
	}
	return nil
}
