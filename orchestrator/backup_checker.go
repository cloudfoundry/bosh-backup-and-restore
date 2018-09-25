package orchestrator

type BackupChecker struct {
	*Workflow
}

func NewBackupChecker(logger Logger, deploymentManager DeploymentManager, lockOrderer LockOrderer) *BackupChecker {
	checkDeployment := NewFindDeploymentStep(deploymentManager, logger)
	backupable := NewBackupableStep(lockOrderer, logger)
	cleanup := NewCleanupStep()
	workflow := NewWorkflow()

	workflow.StartWith(checkDeployment).OnSuccess(backupable)
	workflow.Add(backupable).OnSuccessOrFailure(cleanup)
	workflow.Add(cleanup)

	return &BackupChecker{
		Workflow: workflow,
	}
}

func (b BackupChecker) Check(deploymentName string) Error {
	session := NewSession(deploymentName)

	err := b.Workflow.Run(session)

	return err
}
