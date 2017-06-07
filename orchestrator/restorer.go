package orchestrator

import (
	"fmt"

	"github.com/pkg/errors"
)

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

func (b Restorer) Restore(deploymentName, artifactPath string) Error {
	b.Logger.Info("bbr", "Starting restore of %s...\n", deploymentName)
	backup, err := b.BackupManager.Open(artifactPath, b.Logger)
	if err != nil {
		return Error{errors.Wrap(err, "Could not open backup")}
	}

	if valid, err := backup.Valid(); err != nil {
		return Error{errors.Wrap(err, "Could not validate backup")}
	} else if !valid {
		return Error{errors.Errorf("Backup is corrupted")}
	}

	deployment, err := b.DeploymentManager.Find(deploymentName)
	if err != nil {
		return Error{errors.Wrap(err, "Couldn't find deployment")}
	}

	if !deployment.IsRestorable() {
		return cleanupAndReturnErrors(deployment, errors.Errorf("Deployment '%s' has no restore scripts", deploymentName))
	}

	if match, err := backup.DeploymentMatches(deploymentName, deployment.Instances()); err != nil {
		return cleanupAndReturnErrors(deployment, errors.Errorf("Unable to check if deployment '%s' matches the structure of the provided backup", deploymentName))
	} else if match != true {
		return cleanupAndReturnErrors(deployment, errors.Errorf("Deployment '%s' does not match the structure of the provided backup", deploymentName))
	}

	err = deployment.CheckArtifactDir()
	if err != nil {
		return cleanupAndReturnErrors(deployment, errors.Wrap(err, "Check artifact dir failed"))
	}

	if err = deployment.CopyLocalBackupToRemote(backup); err != nil {
		return cleanupAndReturnErrors(deployment, errors.Errorf("Unable to send backup to remote machine. Got error: %s", err))
	}

	err = deployment.Restore()
	if err != nil {
		return cleanupAndReturnErrors(deployment, errors.Wrap(err, "Failed to restore"))
	}

	b.Logger.Info("bbr", "Completed restore of %s\n", deploymentName)

	if err := deployment.Cleanup(); err != nil {
		return Error{
			NewCleanupError(
				fmt.Sprintf("Deployment '%s' failed while cleaning up with error %v", deploymentName, err),
			),
		}
	}
	return nil
}
