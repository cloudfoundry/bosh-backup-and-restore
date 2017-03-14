package orchestrator

import "fmt"

type Restorer struct {
	BoshClient
	ArtifactManager
	Logger

	DeploymentManager
}

func NewRestorer(bosh BoshClient, artifactManager ArtifactManager, logger Logger, deploymentManager DeploymentManager) *Restorer {
	return &Restorer{
		BoshClient:        bosh,
		ArtifactManager:   artifactManager,
		Logger:            logger,
		DeploymentManager: deploymentManager,
	}
}

func (b Restorer) Restore(deploymentName string) error {
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

	if !deployment.IsRestorable() {
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
