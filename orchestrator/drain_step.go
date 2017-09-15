package orchestrator

import "time"

type DrainStep struct {
	logger Logger
}

func NewDrainStep(logger Logger) Step {
	return &DrainStep{
		logger: logger,
	}
}

func (s *DrainStep) Run(session *Session) error {
	defer s.logger.Info("bbr", "Backup created of %s on %v\n", session.DeploymentName(), time.Now())
	return session.CurrentDeployment().CopyRemoteBackupToLocal(session.CurrentArtifact())

}
