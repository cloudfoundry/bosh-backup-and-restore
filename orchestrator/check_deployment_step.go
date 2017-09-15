package orchestrator

type FindDeploymentStep struct {
	deploymentManager DeploymentManager
	logger            Logger
}

func NewFindDeploymentStep(deploymentManager DeploymentManager, logger Logger) Step {
	return &FindDeploymentStep{deploymentManager: deploymentManager, logger: logger}
}

func (s *FindDeploymentStep) Run(session *Session) error {
	s.logger.Info("bbr", "Running pre-checks for backup of %s...\n", session.DeploymentName())

	s.logger.Info("bbr", "Scripts found:")
	deployment, err := s.deploymentManager.Find(session.DeploymentName())
	if err != nil {
		return err
	}

	session.SetCurrentDeployment(deployment)

	return nil
}
