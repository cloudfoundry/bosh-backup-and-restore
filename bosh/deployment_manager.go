package bosh

import "github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"

func NewBoshDeploymentManager(boshDirector BoshClient, logger Logger, downloadManifest bool) *BoshDeploymentManager {
	return &BoshDeploymentManager{BoshClient: boshDirector, Logger: logger, downloadManifest: downloadManifest}
}

type BoshDeploymentManager struct {
	BoshClient
	Logger
	downloadManifest bool
}

func (b *BoshDeploymentManager) Find(deploymentName string) (orchestrator.Deployment, error) {
	instances, err := b.FindInstances(deploymentName)
	return orchestrator.NewDeployment(b.Logger, instances), err
}

func (b *BoshDeploymentManager) SaveManifest(deploymentName string, artifact orchestrator.Artifact) error {
	if b.downloadManifest {
		manifest, err := b.GetManifest(deploymentName)
		if err != nil {
			return err
		}

		return artifact.SaveManifest(manifest)
	}

	return nil
}
