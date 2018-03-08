package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
)

func BuildDeploymentRestoreCleanuper(target,
	usename,
	password,
	caCert string,
	withManifest,
	isDebug bool) (*orchestrator.RestoreCleaner, error) {

	logger := BuildLogger(isDebug)

	deploymentManager, err := BuildBoshDeploymentManager(
		target,
		usename,
		password,
		caCert,
		logger,
		withManifest,
	)

	if err != nil {
		return nil, err
	}

	return orchestrator.NewRestoreCleaner(logger, deploymentManager, orderer.NewKahnRestoreLockOrderer(), executor.NewParallelExecutor()), nil
}
