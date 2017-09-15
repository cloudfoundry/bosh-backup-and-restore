package orchestrator

type Restorer struct {
	BackupManager
	Logger

	DeploymentManager
}

func NewRestorer(backupManager BackupManager, logger Logger, deploymentManager DeploymentManager) *Restorer {
	return &Restorer{
		BackupManager:     backupManager,
		Logger:            logger,
		DeploymentManager: deploymentManager,
	}
}

func (r Restorer) Restore(deploymentName, backupPath string) Error {
	session := NewSession(deploymentName)
	session.SetCurrentArtifactPath(backupPath)
	workflow := r.buildRestoreWorkflow()
	err := workflow.Run(session)

	return err
}

func (r Restorer) buildRestoreWorkflow() *Workflow {
	workflow := NewWorkflow()

	validateArtifactStep := NewValidateArtifactStep(r.Logger, r.BackupManager)
	checkDeploymentStep := NewCheckDeploymentStep(r.DeploymentManager, r.Logger)
	restorableStep := NewRestorableStep()
	cleanupStep := NewCleanupStep()
	copyToRemoteStep := NewCopyToRemoteStep()
	preRestoreLockStep := NewPreRestoreLockStep()
	restoreStep := NewRestoreStep(r.Logger)
	postRestoreUnlockStep := NewPostRestoreUnlockStep()

	workflow.StartWith(validateArtifactStep).OnSuccess(checkDeploymentStep)
	workflow.Add(checkDeploymentStep).OnSuccess(restorableStep)
	workflow.Add(restorableStep).OnSuccess(copyToRemoteStep).OnFailure(cleanupStep)
	workflow.Add(copyToRemoteStep).OnSuccess(preRestoreLockStep).OnFailure(cleanupStep)
	workflow.Add(preRestoreLockStep).OnSuccess(restoreStep).OnFailure(postRestoreUnlockStep)
	workflow.Add(restoreStep).OnSuccess(postRestoreUnlockStep).OnFailure(postRestoreUnlockStep)
	workflow.Add(postRestoreUnlockStep).OnSuccessOrFailure(cleanupStep)
	workflow.Add(cleanupStep)

	return workflow
}
