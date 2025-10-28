package orchestrator

import "github.com/cloudfoundry/bosh-backup-and-restore/executor"

func NewRestoreCleaner(logger Logger, deploymentManager DeploymentManager, lockOrderer LockOrderer, executor executor.Executor) *RestoreCleaner {
	workflow := NewWorkflow()
	findDeploymentStep := NewFindDeploymentStep(deploymentManager, logger)
	postRestoreUnlockStep := NewPostRestoreUnlockStep(lockOrderer, executor)
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
	currentError := c.Workflow.Run(session) //nolint:staticcheck

	if len(currentError) == 0 {
		c.Logger.Info("bbr", "'%s' cleaned up\n", deploymentName) //nolint:staticcheck
	}
	return currentError
}
