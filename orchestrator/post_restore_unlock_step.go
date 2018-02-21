package orchestrator

type PostRestoreUnlockStep struct {
	lockOrderer LockOrderer
	jobExecutionStategy   JobExecutionStrategy
}

func NewPostRestoreUnlockStep(lockOrderer LockOrderer, jobExecutionStategy JobExecutionStrategy) Step {
	return &PostRestoreUnlockStep{
		lockOrderer: lockOrderer,
		jobExecutionStategy:   jobExecutionStategy,
	}
}

func (s *PostRestoreUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostRestoreUnlock(s.lockOrderer, s.jobExecutionStategy)

	if err != nil {
		return NewPostUnlockError(err.Error())
	}

	return nil
}
