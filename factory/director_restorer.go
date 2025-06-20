package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ratelimiter"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/standalone"
)

func BuildDirectorRestorer(host, username, privateKeyPath, bbrVersion string, hasDebug bool, rateLimiter ratelimiter.RateLimiter) *orchestrator.Restorer {
	logger := BuildLogger(hasDebug)
	deploymentManager := standalone.NewDeploymentManager(logger,
		host,
		username,
		privateKeyPath,
		instance.NewJobFinderOmitMetadataReleases(bbrVersion, logger),
		ssh.NewSshRemoteRunner,
		rateLimiter,
	)

	return orchestrator.NewRestorer(
		backup.BackupDirectoryManager{},
		logger,
		deploymentManager,
		orderer.NewKahnRestoreLockOrderer(),
		executor.NewSerialExecutor(),
		orchestrator.NewArtifactCopier(executor.NewParallelExecutor(), logger),
	)
}
