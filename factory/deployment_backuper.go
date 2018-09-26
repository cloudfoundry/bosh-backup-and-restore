package factory

import (
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func BuildDeploymentBackuper(withManifest bool, boshClient bosh.Client, logger boshlog.Logger) *orchestrator.Backuper {

	execr := executor.NewParallelExecutor()

	return orchestrator.NewBackuper(
		backup.BackupDirectoryManager{},
		logger,
		bosh.NewDeploymentManager(boshClient, logger, withManifest),
		orderer.NewKahnBackupLockOrderer(),
		execr,
		time.Now,
		orchestrator.NewArtifactCopier(execr, logger),
	)
}
