package orchestrator

type FindDeploymentStep struct {
	deploymentManager DeploymentManager
	logger            Logger
}

func NewFindDeploymentStep(deploymentManager DeploymentManager, logger Logger) Step {
	return &FindDeploymentStep{deploymentManager: deploymentManager, logger: logger}
}

func (s *FindDeploymentStep) Run(session *Session) error {
	s.logger.Info("bbr", "Looking for scripts")
	deployment, err := s.deploymentManager.Find(session.DeploymentName())
	if err != nil {
		return err
	}

	session.SetCurrentDeployment(deployment)

	return nil
}
