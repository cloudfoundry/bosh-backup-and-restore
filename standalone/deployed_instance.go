package standalone

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

type DeployedInstance struct {
	*instance.DeployedInstance
}

func NewDeployedInstance(instanceGroupName string, connection ssh.SSHConnection, logger instance.Logger, jobs orchestrator.Jobs, artifactDirCreated bool) DeployedInstance {
	return DeployedInstance{
		DeployedInstance: instance.NewDeployedInstance("0", instanceGroupName, "0", artifactDirCreated, connection, logger, jobs),
	}
}

func (i DeployedInstance) Cleanup() error {
	if !i.ArtifactDirCreated() {
		i.Logger.Debug("", "Backup directory was never created - skipping cleanup")
		return nil
	}

	return i.cleanupArtifact()
}

func (i DeployedInstance) CleanupPrevious() error {
	return i.cleanupArtifact()
}

func (i DeployedInstance) cleanupArtifact() error{
	i.Logger.Info("", "Cleaning up...")

	_, _, exitCode, err := i.RunOnInstance(fmt.Sprintf("sudo rm -rf %s", orchestrator.ArtifactDirectory), "cleanup backup artifacts")

	if err != nil {
		i.Logger.Error("", "Backup artifact clean up failed")
		return errors.Wrap(err, "standalone.DeployedInstance.Cleanup failed")
	}

	if exitCode != 0 {
		return errors.New("Unable to clean up backup artifact")
	}

	return nil
}
