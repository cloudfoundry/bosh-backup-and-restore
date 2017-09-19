package standalone

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"

	"io/ioutil"

	gossh "golang.org/x/crypto/ssh"
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
		return nil, errors.Wrap(err, "failed reading private key")
	}

	connection, err := dm.connectionFactory(dm.hostName, dm.username, string(keyContents), gossh.InsecureIgnoreHostKey(), nil, dm.Logger)
	if err != nil {
		return nil, err
	}

	//TODO: change instanceIdentifier, its not always bosh
	instanceIdentifier := instance.InstanceIdentifier{InstanceGroupName: "bosh", InstanceId: "0"}
	jobs, err := dm.jobFinder.FindJobs(instanceIdentifier, connection, instance.NoopReleaseMapping())
	if err != nil {
		return nil, err
	}

	return orchestrator.NewDeployment(dm.Logger, []orchestrator.Instance{
		NewDeployedInstance("bosh", connection, dm.Logger, jobs, false),
	}), nil
}

func (DeploymentManager) SaveManifest(deploymentName string, artifact orchestrator.Backup) error {
	return nil
}
