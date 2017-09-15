package orchestrator

type LockStep struct {
	lockOrderer LockOrderer
}

func (s *LockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PreBackupLock(s.lockOrderer)
	if err != nil {
		return NewLockError(err.Error())
	}
	return nil
}

func NewLockStep(lockOrderer LockOrderer) Step {
	return &LockStep{lockOrderer: lockOrderer}
}
