package backuper

import (
	"fmt"
	"time"
)

func New(bosh BoshDirector, artifactManager ArtifactManager, logger Logger, deploymentManager DeploymentManager) *Backuper {
	return &Backuper{
		BoshDirector:      bosh,
		ArtifactManager:   artifactManager,
		Logger:            logger,
		DeploymentManager: deploymentManager,
	}
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Warn(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

type Backuper struct {
	BoshDirector
	ArtifactManager
	Logger

	DeploymentManager
}

//go:generate counterfeiter -o fakes/fake_bosh_director.go . BoshDirector
type BoshDirector interface {
	FindInstances(deploymentName string) ([]Instance, error)
	GetManifest(deploymentName string) (string, error)
}

type CleanupError struct {
	error
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) error {
	b.Logger.Info("", "Starting backup of %s...\n", deploymentName)

	exists := b.ArtifactManager.Exists(deploymentName)
	if exists {
		return fmt.Errorf("artifact %s already exists", deploymentName)
	}

	deployment, err := b.DeploymentManager.Find(deploymentName)
	if err != nil {
		return err
	}

	cleanupAndReturnErrors := func(err error) error {
		deployment.Cleanup()
		return err
	}

	if backupable, err := deployment.IsBackupable(); err != nil {
		return cleanupAndReturnErrors(err)
	} else if !backupable {
		return cleanupAndReturnErrors(fmt.Errorf("Deployment '%s' has no backup scripts", deploymentName))
	}

	artifact, err := b.ArtifactManager.Create(deploymentName, b.Logger)
	if err != nil {
		return cleanupAndReturnErrors(err)
	}
	manifest, err := b.GetManifest(deploymentName)
	if err != nil {
		return cleanupAndReturnErrors(err)
	}
	//TODO: Handle this error
	artifact.SaveManifest(manifest)

	if err = deployment.Backup(); err != nil {
		return cleanupAndReturnErrors(err)
	}

	if err = deployment.CopyRemoteBackupToLocal(artifact); err != nil {
		return cleanupAndReturnErrors(err)
	}

	b.Logger.Info("", "Backup created of %s on %v\n", deploymentName, time.Now())

	if err := deployment.Cleanup(); err != nil {
		return CleanupError{
			fmt.Errorf("Deployment '%s' failed while cleaning up with error: %v", deploymentName, err),
		}
	}
	return nil
}

func (b Backuper) Restore(deploymentName string) error {
	b.Logger.Info("", "Starting restore of %s...\n", deploymentName)
	artifact, err := b.ArtifactManager.Open(deploymentName, b.Logger)
	if err != nil {
		return err
	}

	if valid, err := artifact.Valid(); err != nil {
		return err
	} else if !valid {
		return fmt.Errorf("Backup artifact is corrupted")
	}

	deployment, err := b.DeploymentManager.Find(deploymentName)
	if err != nil {
		return err
	}

	defer deployment.Cleanup()

	if restoreable, err := deployment.IsRestorable(); err != nil {
		return err
	} else if !restoreable {
		return fmt.Errorf("Deployment '%s' has no restore scripts", deploymentName)
	}

	if match, err := artifact.DeploymentMatches(deploymentName, deployment.Instances()); err != nil {
		return fmt.Errorf("Unable to check if deployment '%s' matches the structure of the provided backup", deploymentName)
	} else if match != true {
		return fmt.Errorf("Deployment '%s' does not match the structure of the provided backup", deploymentName)
	}

	if err = deployment.CopyLocalBackupToRemote(artifact); err != nil {
		return fmt.Errorf("Unable to send backup to remote machine. Got error: %s", err)
	}

	err = deployment.Restore()
	if err != nil {
		return err
	}

	b.Logger.Info("", "Completed restore of %s\n", deploymentName)
	return nil
}
