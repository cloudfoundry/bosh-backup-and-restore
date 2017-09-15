package orchestrator

import "github.com/pkg/errors"

type RestoreStep struct {
	logger Logger
}

func NewRestoreStep(logger Logger) Step {
	return &RestoreStep{logger: logger}
}

func (s *RestoreStep) Run(session *Session) error {
	err := session.CurrentDeployment().Restore()

	if err != nil {
		return errors.Wrap(err, "Failed to restore")
	}

	s.logger.Info("bbr", "Completed restore of %s\n", session.DeploymentName())
	return nil
}
