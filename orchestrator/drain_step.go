package orchestrator

import "time"

type DrainStep struct {
	logger   Logger
	executor Executor
}

func NewDrainStep(logger Logger, executor Executor) Step {
	return &DrainStep{
		logger:   logger,
		executor: executor,
	}
}

func (s *DrainStep) Run(session *Session) error {
	defer s.logger.Info("bbr", "Backup created of %s on %v\n", session.DeploymentName(), time.Now())
	return session.CurrentDeployment().CopyRemoteBackupToLocal(session.CurrentArtifact(), s.executor)
}
