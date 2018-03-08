package orchestrator

type PostRestoreUnlockStep struct {
	lockOrderer LockOrderer
	executor    Executor
}

func NewPostRestoreUnlockStep(lockOrderer LockOrderer, executor Executor) Step {
	return &PostRestoreUnlockStep{
		lockOrderer: lockOrderer,
		executor:    executor,
	}
}

func (s *PostRestoreUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostRestoreUnlock(s.lockOrderer, s.executor)

	if err != nil {
		return NewPostUnlockError(err.Error())
	}

	return nil
}
