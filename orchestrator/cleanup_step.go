package orchestrator

import "fmt"

type CleanupStep struct{}

func NewCleanupStep() Step {
	return &CleanupStep{}
}

func (s *CleanupStep) Run(session *Session) error {

	if err := session.CurrentDeployment().Cleanup(); err != nil {
		return NewCleanupError(
			fmt.Sprintf("Deployment '%s' failed while cleaning up with error: %v", session.DeploymentName(), err))
	}
	return nil
}
