package orchestrator

import "time"

func NewBackuper(artifactManager ArtifactManager, logger Logger, deploymentManager DeploymentManager, nowFunc func() time.Time) *Backuper {
	return &Backuper{
		ArtifactManager:   artifactManager,
		Logger:            logger,
		DeploymentManager: deploymentManager,
		NowFunc:           nowFunc,
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
	SaveManifest(deploymentName string, artifact Artifact) error
}

type Backuper struct {
	ArtifactManager
	Logger

	DeploymentManager
	NowFunc func() time.Time
}

type AuthInfo struct {
	Type   string
	UaaUrl string
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) Error {
	bw := newBackupWorkflow(b, deploymentName)

	return bw.Run()
}

func (b Backuper) CanBeBackedUp(deploymentName string) (bool, Error) {
	bw := newBackupCheckWorkflow(b, deploymentName)

	err := bw.Run()
	return err == nil, err
}

func cleanupAndReturnErrors(d Deployment, err error) Error {
	cleanupErr := d.Cleanup()
	if cleanupErr != nil {
		return Error{cleanupErr, err}
	}
	return Error{err}
}
