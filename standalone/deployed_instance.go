package standalone

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

type DeployedInstance struct {
	*instance.DeployedInstance
}

func NewDeployedInstance(instanceGroupName string, remoteRunner ssh.RemoteRunner, logger instance.Logger, jobs orchestrator.Jobs, artifactDirCreated bool) DeployedInstance {
	return DeployedInstance{
		DeployedInstance: instance.NewDeployedInstance("0", instanceGroupName, "0", artifactDirCreated, remoteRunner, logger, jobs),
	}
}

func (i DeployedInstance) Cleanup() error {
	if !i.ArtifactDirCreated() {
		i.Logger.Debug("bbr", "Backup directory was never created - skipping cleanup") //nolint:staticcheck
		return nil
	}

	return i.cleanupArtifact()
}

func (i DeployedInstance) CleanupPrevious() error {
	return i.cleanupArtifact()
}

func (i DeployedInstance) cleanupArtifact() error {
	i.Logger.Info("bbr", "Cleaning up...") //nolint:staticcheck

	err := i.RemoveArtifactDir()
	if err != nil {
		i.Logger.Error("bbr", "Backup artifact clean up failed") //nolint:staticcheck
		return errors.Wrap(err, "Unable to clean up backup artifact")
	}

	return nil
}
