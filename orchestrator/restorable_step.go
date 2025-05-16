package orchestrator

import (
	"fmt"

	"github.com/pkg/errors"
)

type RestorableStep struct {
	lockOrderer LockOrderer
	logger      Logger
}

func NewRestorableStep(lockOrderer LockOrderer, logger Logger) Step {
	return &RestorableStep{
		lockOrderer: lockOrderer,
		logger:      logger,
	}
}

func (s *RestorableStep) Run(session *Session) error {

	for _, instance := range session.CurrentDeployment().RestorableInstances() {
		if instance.HasMetadataRestoreNames() {
			errMsg := fmt.Sprintf("discontinued metadata keys backup_name/restore_name found on instance %s. bbr cannot restore this backup artifact.", instance.Name())
			s.logger.Error("bbr", errMsg)
			return errors.New(errMsg)
		}
	}

	if !session.CurrentDeployment().IsRestorable() {
		return errors.Errorf("Deployment '%s' has no restore scripts", session.DeploymentName())
	}

	if match, err := session.CurrentArtifact().DeploymentMatches(session.DeploymentName(), session.CurrentDeployment().Instances()); err != nil {
		return errors.Errorf("Unable to check if deployment '%s' matches the structure of the provided backup", session.DeploymentName())
	} else if match != true { //nolint:staticcheck
		return errors.Errorf("Deployment '%s' does not match the structure of the provided backup", session.DeploymentName())
	}

	err := session.CurrentDeployment().CheckArtifactDir()
	if err != nil {
		return errors.Wrap(err, "Check artifact dir failed")
	}

	if err := session.CurrentDeployment().ValidateLockingDependencies(s.lockOrderer); err != nil {
		return err
	}

	return nil
}
