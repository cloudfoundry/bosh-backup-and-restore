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
	hostName            string
	username            string
	privateKeyFile      string
	jobFinder           instance.JobFinder
	remoteRunnerFactory ssh.RemoteRunnerFactory
}

func NewDeploymentManager(
	logger orchestrator.Logger,
	hostName, username, privateKey string,
	jobFinder instance.JobFinder,
	remoteRunnerFactory ssh.RemoteRunnerFactory,
) DeploymentManager {
	return DeploymentManager{
		Logger:              logger,
		hostName:            hostName,
		username:            username,
		privateKeyFile:      privateKey,
		jobFinder:           jobFinder,
		remoteRunnerFactory: remoteRunnerFactory,
	}
}

func (dm DeploymentManager) Find(deploymentName string) (orchestrator.Deployment, error) {
	keyContents, err := ioutil.ReadFile(dm.privateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading private key")
	}

	remoteRunner, err := dm.remoteRunnerFactory(dm.hostName, dm.username, string(keyContents), gossh.InsecureIgnoreHostKey(), nil, dm.Logger)
	if err != nil {
		return nil, err
	}

	instanceIdentifier := instance.InstanceIdentifier{InstanceGroupName: "bosh", InstanceId: "0"}

	//TODO: change instanceIdentifier, its not always bosh
	jobs, err := dm.jobFinder.FindJobs(instanceIdentifier, remoteRunner, instance.NewNoopManifestQuerier())
	if err != nil {
		return nil, err
	}

	return orchestrator.NewDeployment(dm.Logger, []orchestrator.Instance{
		NewDeployedInstance("bosh", remoteRunner, dm.Logger, jobs, false),
	}), nil
}

func (DeploymentManager) SaveManifest(deploymentName string, artifact orchestrator.Backup) error {
	return nil
}
