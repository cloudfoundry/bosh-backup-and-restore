package orchestrator

import "github.com/cloudfoundry/bosh-backup-and-restore/executor"

type BackupStep struct {
	executor executor.Executor
}

func (s *BackupStep) Run(session *Session) error {
	err := session.CurrentDeployment().Backup(s.executor)
	if err != nil {
		return NewBackupError(err.Error())
	}
	return nil
}

func NewBackupStep(executor executor.Executor) Step {
	return &BackupStep{
		executor: executor,
	}
}
