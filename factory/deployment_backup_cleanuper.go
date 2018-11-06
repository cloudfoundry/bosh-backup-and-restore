package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
	"github.com/cloudfoundry/bosh-utils/logger"
)

func BuildDeploymentBackupCleanuper(
	target string,
	username string,
	password string,
	caCert string,
	hasManifest bool,
	logger logger.Logger,
) (*orchestrator.BackupCleaner, error) {

	boshClient, err := BuildBoshClient(target, username, password, caCert, logger)

	if err != nil {
		return nil, err
	}

	return orchestrator.NewBackupCleaner(logger,
		bosh.NewDeploymentManager(boshClient, logger, hasManifest), orderer.NewKahnBackupLockOrderer(),
		executor.NewParallelExecutor()), nil
}
