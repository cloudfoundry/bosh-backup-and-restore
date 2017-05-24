package standalone

import (
	"fmt"
	"io/ioutil"

	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

type DeploymentManager struct {
	orchestrator.Logger
	hostName          string
	username          string
	privateKeyFile    string
	jobFinder         instance.JobFinder
	connectionFactory ssh.SSHConnectionFactory
}

func NewDeploymentManager(
	logger orchestrator.Logger,
	hostName, username, privateKey string,
	jobFinder instance.JobFinder,
	connectionFactory ssh.SSHConnectionFactory,
) DeploymentManager {
	return DeploymentManager{
		Logger:            logger,
		hostName:          hostName,
		username:          username,
		privateKeyFile:    privateKey,
		jobFinder:         jobFinder,
		connectionFactory: connectionFactory,
	}

}

func (dm DeploymentManager) Find(deploymentName string) (orchestrator.Deployment, error) {
	keyContents, err := ioutil.ReadFile(dm.privateKeyFile)

	if err != nil {
		return nil, err
	}

	connection, err := dm.connectionFactory(dm.hostName, dm.username, string(keyContents), dm.Logger)
	if err != nil {
		return nil, err
	}

	jobs, err := dm.jobFinder.FindJobs("bosh", connection)
	if err != nil {
		return nil, err
	}

	return orchestrator.NewDeployment(dm.Logger, []orchestrator.Instance{
		NewDeployedInstance("bosh", connection, dm.Logger, jobs),
	}), nil
}
func (DeploymentManager) SaveManifest(deploymentName string, artifact orchestrator.Artifact) error {
	return nil
}

type DeployedInstance struct {
	*instance.DeployedInstance
}

func (d DeployedInstance) Cleanup() error {
	d.Logger.Info("", "Cleaning up...")

	stdout, stderr, exitCode, err := d.SSHConnection.Run(fmt.Sprintf("if stat %s; then sudo rm -rf %s; fi", orchestrator.ArtifactDirectory, orchestrator.ArtifactDirectory))
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

func NewDeployedInstance(instanceGroupName string, connection ssh.SSHConnection, logger instance.Logger, jobs instance.Jobs) DeployedInstance {
	return DeployedInstance{
		DeployedInstance: instance.NewDeployedInstance("0", instanceGroupName, "0", connection, logger, jobs),
	}
}
