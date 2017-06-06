package bosh

import (
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

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
	return orchestrator.NewDeployment(b.Logger, instances), errors.Wrap(err, "failed to find instances for deployment "+deploymentName)
}

func (b *DeploymentManager) SaveManifest(deploymentName string, artifact orchestrator.Artifact) error {
	if b.downloadManifest {
		manifest, err := b.GetManifest(deploymentName)
		if err != nil {
			return errors.Wrap(err, "failed to get manifest for deployment "+deploymentName)
		}

		return artifact.SaveManifest(manifest)
	}

	return nil
}
