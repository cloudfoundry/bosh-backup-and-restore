package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
)

func BuildDeploymentBackupCleanuper(
	hasManifest bool,
	boshClient bosh.Client,
	logger bosh.Logger) (*orchestrator.BackupCleaner, error) {
	return orchestrator.NewBackupCleaner(logger,
		bosh.NewDeploymentManager(boshClient, logger, hasManifest), orderer.NewKahnBackupLockOrderer(),
		executor.NewParallelExecutor()), nil
}
