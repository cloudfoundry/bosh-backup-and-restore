package backuper

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	//"github.com/looplab/fsm"
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

func beforeEvent(eventName string) string {
	return "before_" + eventName
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) Error {
	//var artifact Artifact

	b.Logger.Info("", "Starting backup of %s...\n", deploymentName)

	exists := b.ArtifactManager.Exists(deploymentName)
	if exists {
		return Error{fmt.Errorf("artifact %s already exists", deploymentName)}
	}

	deployment, err := b.DeploymentManager.Find(deploymentName)
	if err != nil {
		return Error{err}
	}

	bw := newbackupWorkflow(b, deploymentName, deployment)

	return bw.Run()
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

	if restoreable, err := deployment.IsRestorable(); err != nil {
		return cleanupAndReturnErrors(deployment, err)
	} else if !restoreable {
		return cleanupAndReturnErrors(deployment, fmt.Errorf("Deployment '%s' has no restore scripts", deploymentName))
	}

	if match, err := artifact.DeploymentMatches(deploymentName, deployment.Instances()); err != nil {
		return cleanupAndReturnErrors(deployment, fmt.Errorf("Unable to check if deployment '%s' matches the structure of the provided backup", deploymentName))
	} else if match != true {
		return cleanupAndReturnErrors(deployment, fmt.Errorf("Deployment '%s' does not match the structure of the provided backup", deploymentName))
	}

	if err = deployment.CopyLocalBackupToRemote(artifact); err != nil {
		return cleanupAndReturnErrors(deployment, fmt.Errorf("Unable to send backup to remote machine. Got error: %s", err))
	}

	err = deployment.Restore()
	if err != nil {
		return cleanupAndReturnErrors(deployment, err)
	}

	b.Logger.Info("", "Completed restore of %s\n", deploymentName)

	if err := deployment.Cleanup(); err != nil {
		return CleanupError{
			fmt.Errorf("Deployment '%s' failed while cleaning up with error: %v", deploymentName, err),
		}
	}
	return nil
}

func cleanupAndReturnErrors(d Deployment, err error) error {
	cleanupErr := d.Cleanup()
	if cleanupErr != nil {
		return multierror.Append(err, cleanupErr)
	}
	return err
}

func cleanupAndReturnErrorsArray(d Deployment, err Error) Error {
	cleanupErr := d.Cleanup()
	if cleanupErr != nil {
		return append(err, CleanupError{cleanupErr})
	}
	return err
}
