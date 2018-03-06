package orchestrator

import "time"

type DrainStep struct {
	logger Logger
	executionStrategy ArtifactExecutionStrategy
}

func NewDrainStep(logger Logger, executionStrategy ArtifactExecutionStrategy) Step {
	return &DrainStep{
		logger: logger,
		executionStrategy: executionStrategy,
	}
}

func (s *DrainStep) Run(session *Session) error {
	defer s.logger.Info("bbr", "Backup created of %s on %v\n", session.DeploymentName(), time.Now())
	return session.CurrentDeployment().CopyRemoteBackupToLocal(session.CurrentArtifact(), s.executionStrategy)
}
