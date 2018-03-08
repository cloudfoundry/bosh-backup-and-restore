package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
)

func BuildDeploymentBackupCleanuper(target,
	username,
	password,
	caCert string,
	hasManifest,
	hasDebug bool) (*orchestrator.BackupCleaner, error) {

	logger := BuildLogger(hasDebug)
	deploymentManager, err := BuildBoshDeploymentManager(
		target,
		username,
		password,
		caCert,
		logger,
		hasManifest,
	)

	if err != nil {
		return nil, err
	}

	return orchestrator.NewBackupCleaner(logger, deploymentManager, orderer.NewKahnBackupLockOrderer(),
		executor.NewParallelJobExecutor()), nil
}
