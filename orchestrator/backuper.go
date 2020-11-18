package orchestrator

import (
	"time"

	exe "github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
)

func NewBackuper(backupManager BackupManager, logger Logger, deploymentManager DeploymentManager, lockOrderer LockOrderer,
	executor exe.Executor, nowFunc func() time.Time, artifactCopier ArtifactCopier, unsafeLockFree bool, timestamp string) *Backuper {

	findDeploymentStep := NewFindDeploymentStep(deploymentManager, logger)
	backupable := NewBackupableStep(lockOrderer, logger)
	createArtifact := NewCreateArtifactStep(logger, backupManager, deploymentManager, nowFunc, timestamp)

	backup := NewBackupStep(executor)
	drain := NewDrainStep(logger, artifactCopier)
	cleanup := NewCleanupStep()
	addFinishTimeStep := NewAddFinishTimeStep(nowFunc)

	var lock, unlockAfterSuccessfulBackup, unlockAfterFailedBackup Step
	if !unsafeLockFree {
		lock = NewLockStep(lockOrderer, executor)
		unlockAfterSuccessfulBackup = NewPostBackupUnlockStep(true, lockOrderer, executor)
		unlockAfterFailedBackup = NewPostBackupUnlockStep(false, lockOrderer, executor)
	} else {
		lock = NewSkipStep(logger, "lock")
		unlockAfterSuccessfulBackup = NewSkipStep(logger, "unlock after successful backup")
		unlockAfterFailedBackup = NewSkipStep(logger, "unlock after unsuccessful backup")
	}

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
func (b Backuper) Backup(deploymentName, artifactPath string) Error {
	session := NewSession(deploymentName)
	session.SetCurrentArtifactPath(artifactPath)

	err := b.workflow.Run(session)

	return err
}
