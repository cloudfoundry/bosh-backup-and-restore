package orchestrator

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"

type Restorer struct {
	workflow *Workflow
}

func NewRestorer(backupManager BackupManager, logger Logger, deploymentManager DeploymentManager,
	lockOrderer LockOrderer, executor executor.Executor) *Restorer {
	workflow := NewWorkflow()
	validateArtifactStep := NewValidateArtifactStep(logger, backupManager)
	findDeploymentStep := NewFindDeploymentStep(deploymentManager, logger)
	restorableStep := NewRestorableStep(lockOrderer)
	cleanupStep := NewCleanupStep()
	copyToRemoteStep := NewCopyToRemoteStep(executor)
	preRestoreLockStep := NewPreRestoreLockStep(lockOrderer, executor)
	restoreStep := NewRestoreStep(logger)
	postRestoreUnlockStep := NewPostRestoreUnlockStep(lockOrderer, executor)

	workflow.StartWith(validateArtifactStep).OnSuccess(findDeploymentStep)
	workflow.Add(findDeploymentStep).OnSuccess(restorableStep)
	workflow.Add(restorableStep).OnSuccess(copyToRemoteStep).OnFailure(cleanupStep)
	workflow.Add(copyToRemoteStep).OnSuccess(preRestoreLockStep).OnFailure(cleanupStep)
	workflow.Add(preRestoreLockStep).OnSuccess(restoreStep).OnFailure(postRestoreUnlockStep)
	workflow.Add(restoreStep).OnSuccess(postRestoreUnlockStep).OnFailure(postRestoreUnlockStep)
	workflow.Add(postRestoreUnlockStep).OnSuccessOrFailure(cleanupStep)
	workflow.Add(cleanupStep)
	return &Restorer{
		workflow: workflow,
	}
}

func (r Restorer) Restore(deploymentName, backupPath string) Error {
	session := NewSession(deploymentName)
	session.SetCurrentArtifactPath(backupPath)

	return r.workflow.Run(session)
}
