package orchestrator

func NewRestoreCleaner(logger Logger, deploymentManager DeploymentManager, lockOrderer LockOrderer) *RestoreCleaner {
	workflow := NewWorkflow()
	findDeploymentStep := NewFindDeploymentStep(deploymentManager, logger)
	postRestoreUnlockStep := NewPostRestoreUnlockStep(lockOrderer)
	cleanupPreviousStep := NewCleanupPreviousStep()

	workflow.StartWith(findDeploymentStep).OnSuccess(postRestoreUnlockStep)
	workflow.Add(postRestoreUnlockStep).OnSuccessOrFailure(cleanupPreviousStep)
	workflow.Add(cleanupPreviousStep)

	return &RestoreCleaner{
		Logger:   logger,
		Workflow: workflow,
	}
}

type RestoreCleaner struct {
	Logger
	*Workflow
}

func (c RestoreCleaner) Cleanup(deploymentName string) Error {
	session := NewSession(deploymentName)
	currentError := c.Workflow.Run(session)

	if len(currentError) == 0 {
		c.Logger.Info("bbr", "'%s' cleaned up\n", deploymentName)
	}
	return currentError
}
