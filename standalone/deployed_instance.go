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

func (d DeployedInstance) Cleanup() error {
	d.Logger.Info("", "Cleaning up...")
	if !d.ArtifactDirCreated() {
		d.Logger.Debug("", "Backup directory was never created - skipping cleanup")
		return nil
	}

	stdout, stderr, exitCode, err := d.SSHConnection.Run(fmt.Sprintf("sudo rm -rf %s", orchestrator.ArtifactDirectory))
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Error("", "Backup artifact clean up failed")
		return errors.Wrap(err, "standalone.DeployedInstance.Cleanup failed")
	}

	if exitCode != 0 {
		return errors.New("Unable to clean up backup artifact")
	}

	return nil
}

func (d DeployedInstance) CleanupPrevious() error {
	d.Logger.Info("", "Cleaning up...")

	stdout, stderr, exitCode, err := d.SSHConnection.Run(fmt.Sprintf("sudo rm -rf %s", orchestrator.ArtifactDirectory))
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Error("", "Backup artifact clean up failed")
		return errors.Wrap(err, "standalone.DeployedInstance.Cleanup failed")
	}

	if exitCode != 0 {
		return errors.New("Unable to clean up backup artifact")
	}

	return nil
}

func NewDeployedInstance(instanceGroupName string, connection ssh.SSHConnection, logger instance.Logger, jobs orchestrator.Jobs, artifactDirCreated bool) DeployedInstance {
	return DeployedInstance{
		DeployedInstance: instance.NewDeployedInstance("0", instanceGroupName, "0", artifactDirCreated, connection, logger, jobs),
	}
}
