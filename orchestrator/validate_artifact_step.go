package orchestrator

import (
	"github.com/pkg/errors"
)

func NewValidateArtifactStep(logger Logger, backupManager BackupManager) Step {
	return &ValidateArtifactStep{logger: logger, backupManager: backupManager}
}

type ValidateArtifactStep struct {
	logger        Logger
	backupManager BackupManager
}

func (s *ValidateArtifactStep) Run(session *Session) error {
	s.logger.Info("bbr", "Starting restore of %s...\n", session.deploymentName)
	backup, err := s.backupManager.Open(session.CurrentArtifactPath(), s.logger)
	if err != nil {
		return errors.Wrap(err, "Could not open backup")
	}
	session.SetCurrentArtifact(backup)

	s.logger.Info("bbr", "Validating backup artifact for %s...\n", session.deploymentName)
	if valid, err := backup.Valid(); err != nil {
		return errors.Wrap(err, "Could not validate backup")
	} else if !valid {
		return errors.Errorf("Backup is corrupted")
	}
	return nil
}
