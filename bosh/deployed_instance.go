package bosh

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
)

type BoshDeployedInstance struct {
	Deployment director.Deployment
	*instance.DeployedInstance
}

func NewBoshDeployedInstance(instanceGroupName,
	instanceIndex,
	instanceID string,
	connection ssh.SSHConnection,
	deployment director.Deployment,
	logger Logger,
	jobs instance.Jobs,
) orchestrator.Instance {
	return &BoshDeployedInstance{
		Deployment:       deployment,
		DeployedInstance: instance.NewDeployedInstance(instanceIndex, instanceGroupName, instanceID, connection, logger, jobs),
	}
}

func (d *BoshDeployedInstance) Cleanup() error {
	var errs []error
	d.Logger.Debug("bbr", "Cleaning up SSH connection on instance %s %s", d.Name(), d.ID())
	removeArtifactError := d.removeBackupArtifacts()
	if removeArtifactError != nil {
		errs = append(errs, removeArtifactError)
	}
	cleanupSSHError := d.Deployment.CleanUpSSH(director.NewAllOrInstanceGroupOrInstanceSlug(d.Name(), d.ID()), director.SSHOpts{Username: d.SSHConnection.Username()})
	if cleanupSSHError != nil {
		errs = append(errs, cleanupSSHError)
	}

	return orchestrator.ConvertErrors(errs)
}
func (d *BoshDeployedInstance) removeBackupArtifacts() error {
	_, _, _, err := d.RunOnInstance(fmt.Sprintf("sudo rm -rf %s", orchestrator.ArtifactDirectory), "remove backup artifacts")
	return err
}
