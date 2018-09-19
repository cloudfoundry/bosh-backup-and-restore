package factory

import (
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
)

func BuildDeploymentBackuper(target, username, password, caCert string, withManifest, hasDebug bool) (*orchestrator.Backuper, error) {
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
	execr := executor.NewParallelExecutor()

	return orchestrator.NewBackuper(
		backup.BackupDirectoryManager{},
		logger,
		bosh.NewDeploymentManager(boshClient, logger, withManifest),
		orderer.NewKahnBackupLockOrderer(),
		execr,
		time.Now,
		orchestrator.NewArtifactCopier(execr, logger),
	), nil
}
