package orchestrator


type LockStep struct {
	lockOrderer LockOrderer
	jobExecutionStategy   JobExecutionStrategy
}

func (s *LockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PreBackupLock(s.lockOrderer, s.jobExecutionStategy)
	if err != nil {
		return NewLockError(err.Error())
	}
	return nil
}

func NewLockStep(lockOrderer LockOrderer, jobExecutionStategy JobExecutionStrategy) Step {
	return &LockStep{lockOrderer: lockOrderer, jobExecutionStategy: jobExecutionStategy}
}
