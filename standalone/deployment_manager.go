package standalone

import (
	"io/ioutil"

	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
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

func (DeployedInstance) Cleanup() error {
	return nil
}

func NewDeployedInstance(instanceGroupName string, connection ssh.SSHConnection, logger instance.Logger, jobs instance.Jobs) DeployedInstance {
	return DeployedInstance{
		DeployedInstance: instance.NewDeployedInstance("0", instanceGroupName, "0", connection, logger, jobs),
	}
}
