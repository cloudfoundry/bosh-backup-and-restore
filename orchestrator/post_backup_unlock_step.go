package orchestrator


type PostBackupUnlockStep struct {
	lockOrderer LockOrderer
	jobExecutionStategy   JobExecutionStrategy
}

func NewPostBackupUnlockStep(lockOrderer LockOrderer, jobExecutionStategy JobExecutionStrategy) Step {
	return &PostBackupUnlockStep{
		lockOrderer: lockOrderer,
		jobExecutionStategy:   jobExecutionStategy,
	}
}

func (s *PostBackupUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostBackupUnlock(s.lockOrderer, s.jobExecutionStategy)
	if err != nil {
		return NewPostUnlockError(err.Error())
	}
	return nil
}
