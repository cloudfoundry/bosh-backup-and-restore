package backuper

import (
	"fmt"
	"time"
)

func New(bosh BoshDirector, artifactCreator ArtifactCreator, logger Logger, deploymentManager DeploymentManager) *Backuper {
	return &Backuper{
		BoshDirector:      bosh,
		ArtifactCreator:   artifactCreator,
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
	ArtifactCreator
	Logger

	DeploymentManager
}

//go:generate counterfeiter -o fakes/fake_bosh_director.go . BoshDirector
type BoshDirector interface {
	FindInstances(deploymentName string) ([]Instance, error)
	GetManifest(deploymentName string) (string, error)
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) error {
	b.Logger.Info("", "Starting backup of %s...\n", deploymentName)
	deployment, err := b.DeploymentManager.Find(deploymentName)
	if err != nil {
		return err
	}

	defer deployment.Cleanup()

	if backupable, err := deployment.IsBackupable(); err != nil {
		return err
	} else if !backupable {
		return fmt.Errorf("Deployment '%s' has no backup scripts", deploymentName)
	}

	artifact, err := b.ArtifactCreator(deploymentName)
	if err != nil {
		return err
	}
	manifest, err := b.GetManifest(deploymentName)
	if err != nil {
		return err
	}
	artifact.SaveManifest(manifest)

	if err = deployment.Backup(); err != nil {
		return err
	}

	if err = deployment.CopyRemoteBackupsToLocalArtifact(artifact); err != nil {
		return err
	}

	b.Logger.Info("", "Backup created of %s on %v\n", deploymentName, time.Now())
	return nil
}

func (b Backuper) Restore(deploymentName string) error {
	b.Logger.Info("", "Starting restore of %s...\n", deploymentName)
	artifact, err := b.ArtifactCreator(deploymentName)
	if err != nil {
		return err
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

	if err = deployment.LoadFrom(artifact); err != nil {
		return fmt.Errorf("Unable to send backup to remote machine. Got error: %s", err)
	}

	err = deployment.Restore()
	if err != nil {
		return err
	}

	b.Logger.Info("", "Completed restore of %s\n", deploymentName)
	return nil
}

func matchChecksums(instance Instance, localChecksum, remoteChecksum map[string]string) error {
	if len(localChecksum) != len(remoteChecksum) {
		return fmt.Errorf("Backup artifact is corrupted, checksum failed for %s:%s, files mismatched, remote files: %d, local files: %d", instance.Name(), instance.ID(), len(remoteChecksum), len(localChecksum))
	}
	for key, _ := range localChecksum {
		if localChecksum[key] != remoteChecksum[key] {
			return fmt.Errorf("Backup artifact is corrupted, checksum failed for %s:%s, checksum mismatched %s, remote file: %s, local file: %s", instance.Name(), instance.ID(), key, remoteChecksum, localChecksum)
		}
	}
	return nil
}
