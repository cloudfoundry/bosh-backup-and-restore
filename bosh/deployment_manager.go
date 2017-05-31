package bosh

import "github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"

func NewDeploymentManager(boshDirector BoshClient, logger Logger, downloadManifest bool) *DeploymentManager {
	return &DeploymentManager{BoshClient: boshDirector, Logger: logger, downloadManifest: downloadManifest}
}

type DeploymentManager struct {
	BoshClient
	Logger
	downloadManifest bool
}

func (b *DeploymentManager) Find(deploymentName string) (orchestrator.Deployment, error) {
	instances, err := b.FindInstances(deploymentName)
	return orchestrator.NewDeployment(b.Logger, instances), err
}

func (b *DeploymentManager) SaveManifest(deploymentName string, artifact orchestrator.Artifact) error {
	if b.downloadManifest {
		manifest, err := b.GetManifest(deploymentName)
		if err != nil {
			return err
		}

		return artifact.SaveManifest(manifest)
	}

	return nil
}
