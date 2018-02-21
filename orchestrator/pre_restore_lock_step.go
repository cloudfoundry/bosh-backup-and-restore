package orchestrator

import "github.com/pkg/errors"

type PreRestoreLockStep struct {
	lockOrderer LockOrderer
	jobExecutionStategy   JobExecutionStrategy
}

func NewPreRestoreLockStep(lockOrderer LockOrderer, jobExecutionStategy JobExecutionStrategy) Step {
	return &PreRestoreLockStep{lockOrderer: lockOrderer,
		jobExecutionStategy: jobExecutionStategy}
}

func (s *PreRestoreLockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PreRestoreLock(s.lockOrderer, s.jobExecutionStategy)

	if err != nil {
		return errors.Wrap(err, "pre-restore-lock failed")
	}
	return nil
}
