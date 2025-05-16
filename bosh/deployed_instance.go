package bosh

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry/bosh-cli/v7/director"
	"github.com/pkg/errors"
)

type BoshDeployedInstance struct {
	Deployment director.Deployment
	*instance.DeployedInstance
}

func NewBoshDeployedInstance(instanceGroupName,
	instanceIndex,
	instanceID string,
	remoteRunner ssh.RemoteRunner,
	deployment director.Deployment,
	artifactDirectoryCreated bool,
	logger Logger,
	jobs orchestrator.Jobs,
) orchestrator.Instance {
	return &BoshDeployedInstance{
		Deployment:       deployment,
		DeployedInstance: instance.NewDeployedInstance(instanceIndex, instanceGroupName, instanceID, artifactDirectoryCreated, remoteRunner, logger, jobs),
	}
}

func (i *BoshDeployedInstance) Cleanup() error {
	var errs []error

	if i.ArtifactDirCreated() {
		removeArtifactError := i.RemoveArtifactDir()
		if removeArtifactError != nil {
			errs = append(errs, errors.Wrap(removeArtifactError, "failed to remove backup artifact"))
		}
	}

	cleanupSSHError := i.cleanupSSHConnections()
	if cleanupSSHError != nil {
		errs = append(errs, errors.Wrap(cleanupSSHError, "failed to cleanup ssh"))
	}

	return orchestrator.ConvertErrors(errs)
}

func (i *BoshDeployedInstance) CleanupPrevious() error {
	var errs []error

	removeArtifactError := i.RemoveArtifactDir()
	if removeArtifactError != nil {
		errs = append(errs, errors.Wrap(removeArtifactError, "failed to remove backup artifact"))
	}

	cleanupSSHError := i.cleanupSSHConnections()
	if cleanupSSHError != nil {
		errs = append(errs, errors.Wrap(cleanupSSHError, "failed to cleanup ssh"))
	}

	return orchestrator.ConvertErrors(errs)
}

func (i *BoshDeployedInstance) cleanupSSHConnections() error {
	i.Logger.Debug("bbr", "Cleaning up SSH connection on instance %s %s", i.Name(), i.ID()) //nolint:staticcheck
	return i.Deployment.CleanUpSSH(director.NewAllOrInstanceGroupOrInstanceSlug(i.Name(), i.ID()), director.SSHOpts{Username: i.ConnectedUsername()})
}
