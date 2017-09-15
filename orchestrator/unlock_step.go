package orchestrator

type UnlockStep struct {
	lockOrderer LockOrderer
}

func NewUnlockStep(lockOrderer LockOrderer) Step {
	return &UnlockStep{
		lockOrderer: lockOrderer,
	}
}

func (s *UnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostBackupUnlock(s.lockOrderer)
	if err != nil {
		return NewPostBackupUnlockError(err.Error())
	}
	return nil
}
