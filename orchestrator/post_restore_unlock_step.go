package orchestrator

type PostRestoreUnlockStep struct {
	lockOrderer LockOrderer
}

func NewPostRestoreUnlockStep(lockOrderer LockOrderer) Step {
	return &PostRestoreUnlockStep{
		lockOrderer: lockOrderer,
	}
}

func (s *PostRestoreUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostRestoreUnlock(s.lockOrderer)

	if err != nil {
		return NewPostUnlockError(err.Error())
	}

	return nil
}
