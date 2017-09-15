package orchestrator

import (
	"time"
)

func NewBackuper(backupManager BackupManager, logger Logger, deploymentManager DeploymentManager, lockOrderer LockOrderer, nowFunc func() time.Time) *Backuper {
	return &Backuper{
		BackupManager:     backupManager,
		Logger:            logger,
		DeploymentManager: deploymentManager,
		NowFunc:           nowFunc,
		LockOrderer:       lockOrderer,
	}
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Warn(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

//go:generate counterfeiter -o fakes/fake_deployment_manager.go . DeploymentManager
type DeploymentManager interface {
	Find(deploymentName string) (Deployment, error)
	SaveManifest(deploymentName string, artifact Backup) error
}

type Backuper struct {
	BackupManager
	Logger
	LockOrderer

	DeploymentManager
	NowFunc func() time.Time
}

type AuthInfo struct {
	Type   string
	UaaUrl string
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) Error {
	session := NewSession(deploymentName)
	workflow := b.buildBackupWorkflow()

	err := workflow.Run(session)

	return err
}

func (b Backuper) buildBackupWorkflow() *Workflow {
	findDeploymentStep := NewFindDeploymentStep(b.DeploymentManager, b.Logger)
	backupable := NewBackupableStep(b.LockOrderer)
	createArtifact := NewCreateArtifactStep(b.Logger, b.BackupManager, b.DeploymentManager, b.NowFunc)
	lock := NewLockStep(b.LockOrderer)
	backup := NewBackupStep()
	unlockAfterSuccessfulBackup := NewPostBackupUnlockStep(b.LockOrderer)
	unlockAfterFailedBackup := NewPostBackupUnlockStep(b.LockOrderer)
	drain := NewDrainStep(b.Logger)
	cleanup := NewCleanupStep()
	addFinishTimeStep := NewAddFinishTimeStep(b.NowFunc)

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

	return workflow
}

func (b Backuper) buildBackupCheckWorkflow() *Workflow {
	checkDeployment := NewFindDeploymentStep(b.DeploymentManager, b.Logger)
	backupable := NewBackupableStep(b.LockOrderer)
	cleanup := NewCleanupStep()
	workflow := NewWorkflow()

	workflow.StartWith(checkDeployment).OnSuccess(backupable)
	workflow.Add(backupable).OnSuccessOrFailure(cleanup)
	workflow.Add(cleanup)

	return workflow
}

func (b Backuper) CanBeBackedUp(deploymentName string) (bool, Error) {
	session := NewSession(deploymentName)
	workflow := b.buildBackupCheckWorkflow()

	err := workflow.Run(session)

	return err == nil, err
}
