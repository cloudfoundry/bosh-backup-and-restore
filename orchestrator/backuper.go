package orchestrator

import (
	"time"
)

func NewBackuper(backupManager BackupManager, logger Logger, deploymentManager DeploymentManager, lockOrderer LockOrderer, nowFunc func() time.Time) *Backuper {
	findDeploymentStep := NewFindDeploymentStep(deploymentManager, logger)
	backupable := NewBackupableStep(lockOrderer, logger)
	createArtifact := NewCreateArtifactStep(logger, backupManager, deploymentManager, nowFunc)
	lock := NewLockStep(lockOrderer)
	backup := NewBackupStep()
	unlockAfterSuccessfulBackup := NewPostBackupUnlockStep(lockOrderer)
	unlockAfterFailedBackup := NewPostBackupUnlockStep(lockOrderer)
	drain := NewDrainStep(logger)
	cleanup := NewCleanupStep()
	addFinishTimeStep := NewAddFinishTimeStep(nowFunc)

	workflow := NewWorkflow()
	workflow.StartWith(findDeploymentStep).OnSuccess(backupable)
	workflow.Add(backupable).OnSuccess(createArtifact).OnFailure(cleanup)
	workflow.Add(createArtifact).OnSuccess(lock).OnFailure(cleanup)
	workflow.Add(lock).OnSuccess(backup).OnFailure(unlockAfterFailedBackup)
	workflow.Add(backup).OnSuccess(unlockAfterSuccessfulBackup).OnFailure(unlockAfterFailedBackup)
	workflow.Add(unlockAfterSuccessfulBackup).OnSuccessOrFailure(drain)
	workflow.Add(unlockAfterFailedBackup).OnSuccessOrFailure(cleanup)
	workflow.Add(drain).OnSuccessOrFailure(cleanup)
	workflow.Add(cleanup).OnSuccessOrFailure(addFinishTimeStep)
	workflow.Add(addFinishTimeStep)

	return &Backuper{
		workflow: workflow,
	}
}

type Backuper struct {
	workflow *Workflow
}

type AuthInfo struct {
	Type   string
	UaaUrl string
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) Error {
	session := NewSession(deploymentName)

	err := b.workflow.Run(session)

	return err
}
