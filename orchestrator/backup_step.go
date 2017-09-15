package orchestrator

type BackupStep struct{}

func (s *BackupStep) Run(session *Session) error {
	err := session.CurrentDeployment().Backup()
	if err != nil {
		return NewBackupError(err.Error())
	}
	return nil
}

func NewBackupStep() Step {
	return &BackupStep{}
}
