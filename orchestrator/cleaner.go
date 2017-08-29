package orchestrator

func NewCleaner(logger Logger, deploymentManager DeploymentManager, lockOrderer LockOrderer) *Cleaner {
	return &Cleaner{
		Logger:            logger,
		DeploymentManager: deploymentManager,
		lockOrderer:       lockOrderer,
	}
}

type Cleaner struct {
	Logger
	DeploymentManager
	lockOrderer LockOrderer
}

func (c Cleaner) Cleanup(deploymentName string) Error {
	deployment, err := c.DeploymentManager.Find(deploymentName)
	if err != nil {
		return Error{err}
	}

	var currentError = Error{}

	err = deployment.PostBackupUnlock(c.lockOrderer)
	if err != nil {
		currentError = append(currentError, err)
	}

	err = deployment.CleanupPrevious()
	if err != nil {
		currentError = append(currentError, err)
	}

	if len(currentError) == 0 {
		c.Logger.Info("bbr", "'%s' cleaned up\n", deploymentName)
	}
	return currentError
}

type NopLockOrderer struct{}

func NewNopLockOrderer() LockOrderer {
	return NopLockOrderer{}
}

func (lo NopLockOrderer) Order(jobs []Job) []Job {
	return jobs
}
