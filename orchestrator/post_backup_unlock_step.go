package orchestrator

type PostBackupUnlockStep struct {
	lockOrderer LockOrderer
}

func NewPostBackupUnlockStep(lockOrderer LockOrderer) Step {
	return &PostBackupUnlockStep{
		lockOrderer: lockOrderer,
	}
}

func (s *PostBackupUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostBackupUnlock(s.lockOrderer)
	if err != nil {
		return NewPostBackupUnlockError(err.Error())
	}
	return nil
}
