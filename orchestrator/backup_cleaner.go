package orchestrator

func NewBackupCleaner(logger Logger, deploymentManager DeploymentManager, lockOrderer LockOrderer) *BackupCleaner {
	workflow := NewWorkflow()
	findDeploymentStep := NewFindDeploymentStep(deploymentManager, logger)
	postBackUnlockStep := NewPostBackupUnlockStep(lockOrderer)
	cleanupPreviousStep := NewCleanupPreviousStep()

	workflow.StartWith(findDeploymentStep).OnSuccess(postBackUnlockStep)
	workflow.Add(postBackUnlockStep).OnSuccessOrFailure(cleanupPreviousStep)
	workflow.Add(cleanupPreviousStep)

	return &BackupCleaner{
		Logger:   logger,
		Workflow: workflow,
	}
}

type BackupCleaner struct {
	Logger
	*Workflow
}

func (c BackupCleaner) Cleanup(deploymentName string) Error {
	session := NewSession(deploymentName)
	currentError := c.Workflow.Run(session)

	if len(currentError) == 0 {
		c.Logger.Info("bbr", "'%s' cleaned up\n", deploymentName)
	}
	return currentError
}
