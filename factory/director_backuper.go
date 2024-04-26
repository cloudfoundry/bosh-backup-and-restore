package factory

import (
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ratelimiter"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/standalone"
)

func BuildDirectorBackuper(host, username, privateKeyPath, bbrVersion string, hasDebug bool, rateLimiter ratelimiter.RateLimiter, timeStamp string) *orchestrator.Backuper {
	logger := BuildLogger(hasDebug)
	deploymentManager := standalone.NewDeploymentManager(logger,
		host,
		username,
		privateKeyPath,
		instance.NewJobFinderOmitMetadataReleases(bbrVersion, logger),
		ssh.NewSshRemoteRunner,
		rateLimiter,
	)
	execr := executor.NewParallelExecutor()

	return orchestrator.NewBackuper(backup.BackupDirectoryManager{}, logger, deploymentManager, orderer.NewKahnBackupLockOrderer(), execr, time.Now, orchestrator.NewArtifactCopier(execr, logger), false, timeStamp)
}
