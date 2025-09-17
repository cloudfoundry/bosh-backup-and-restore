package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ratelimiter"
)

func BuildDeploymentRestorer(target, username, password, caCert, bbrVersion string, debug bool, rateLimiter ratelimiter.RateLimiter) (*orchestrator.Restorer, error) {
	logger := BuildLogger(debug)
	boshClient, err := BuildBoshClient(
		target,
		username,
		password,
		caCert,
		bbrVersion,
		rateLimiter,
		logger,
	)
	if err != nil {
		return nil, err
	}

	return orchestrator.NewRestorer(
		backup.BackupDirectoryManager{},
		logger,
		bosh.NewDeploymentManager(boshClient, logger, false),
		orderer.NewKahnRestoreLockOrderer(),
		executor.NewSerialExecutor(),
		orchestrator.NewArtifactCopier(executor.NewParallelExecutor(), logger),
	), nil
}
