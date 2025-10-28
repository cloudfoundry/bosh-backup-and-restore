package factory

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry/bosh-backup-and-restore/orderer"
	"github.com/cloudfoundry/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry/bosh-backup-and-restore/standalone"
)

func BuildDirectorBackupCleaner(host,
	username,
	privateKeyPath,
	bbrVersion string,
	hasDebug bool) *orchestrator.BackupCleaner {

	logger := BuildLogger(hasDebug)
	deploymentManager := standalone.NewDeploymentManager(logger,
		host,
		username,
		privateKeyPath,
		instance.NewJobFinderOmitMetadataReleases(bbrVersion, logger),
		ssh.NewSshRemoteRunner,
	)

	return orchestrator.NewBackupCleaner(logger, deploymentManager, orderer.NewKahnBackupLockOrderer(), executor.NewParallelExecutor())
}
