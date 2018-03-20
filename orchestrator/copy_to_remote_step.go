package orchestrator

import (
	"github.com/pkg/errors"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
)

type CopyToRemoteStep struct {
	executor executor.Executor
}

func NewCopyToRemoteStep(executor executor.Executor) Step {
	return &CopyToRemoteStep{
		executor: executor,
	}
}

func (s *CopyToRemoteStep) Run(session *Session) error {
	if err := session.CurrentDeployment().CopyLocalBackupToRemote(session.CurrentArtifact(), s.executor); err != nil {
		return errors.Errorf("Unable to send backup to remote machine. Got error: %s", err)
	}
	return nil
}
