package factory

import (
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/artifactexecutor"
)

func BuildDeploymentBackuper(target, username, password, caCert string, withManifest, hasDebug bool) (*orchestrator.Backuper, error) {
	logger := BuildLogger(hasDebug)
	deploymentManager, err := BuildBoshDeploymentManager(
		target,
		username,
		password,
		caCert,
		logger,
		withManifest,
	)

	if err != nil {
		return nil, err
	}

	return orchestrator.NewBackuper(backup.BackupDirectoryManager{}, logger, deploymentManager,
		orderer.NewKahnBackupLockOrderer(), executor.NewParallelJobExecutor(), artifactexecutor.NewParallelExecutionStrategy(), time.Now), nil
}
