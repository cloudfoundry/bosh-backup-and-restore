package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
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
	boshClient, err := BuildBoshClient(
		target,
		username,
		password,
		caCert,
		logger,
	)

	if err != nil {
		return nil, err
	}

	return orchestrator.NewBackupCleaner(logger,
		bosh.NewDeploymentManager(boshClient, logger, hasManifest), orderer.NewKahnBackupLockOrderer(),
		executor.NewParallelExecutor()), nil
}
