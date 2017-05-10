package bosh

import "github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"

func NewBoshDeploymentManager(boshDirector BoshClient, logger Logger) *BoshDeploymentManager {
	return &BoshDeploymentManager{BoshClient: boshDirector, Logger: logger}
}

type BoshDeploymentManager struct {
	BoshClient
	Logger
}

func (b *BoshDeploymentManager) Find(deploymentName string) (orchestrator.Deployment, error) {
	instances, err := b.FindInstances(deploymentName)
	return orchestrator.NewDeployment(b.Logger, instances), err
}
func (b *BoshDeploymentManager) SaveManifest(deploymentName string, artifact orchestrator.Artifact) error {
	manifest, err := b.GetManifest(deploymentName)
	if err != nil {
		return err
	}

	return artifact.SaveManifest(manifest)
}
