package orchestrator

import (
	"github.com/pkg/errors"
)

type RestorableStep struct {
	lockOrderer LockOrderer
}

func NewRestorableStep(lockOrderer LockOrderer) Step {
	return &RestorableStep{
		lockOrderer: lockOrderer,
	}
}

func (s *RestorableStep) Run(session *Session) error {
	if !session.CurrentDeployment().IsRestorable() {
		return errors.Errorf("Deployment '%s' has no restore scripts", session.DeploymentName())
	}

	if match, err := session.CurrentArtifact().DeploymentMatches(session.DeploymentName(), session.CurrentDeployment().Instances()); err != nil {
		return errors.Errorf("Unable to check if deployment '%s' matches the structure of the provided backup", session.DeploymentName())
	} else if match != true {
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
