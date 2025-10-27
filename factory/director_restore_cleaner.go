package factory

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry/bosh-backup-and-restore/orderer"
	"github.com/cloudfoundry/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry/bosh-backup-and-restore/standalone"
)

func BuildDirectorRestoreCleaner(host,
	username,
	privateKeyPath,
	bbrVersion string,
	hasDebug bool) *orchestrator.RestoreCleaner {

	logger := BuildLogger(hasDebug)

	deploymentManager := standalone.NewDeploymentManager(logger,
		host,
		username,
		privateKeyPath,
		instance.NewJobFinderOmitMetadataReleases(bbrVersion, logger),
		ssh.NewSshRemoteRunner,
	)

	return orchestrator.NewRestoreCleaner(logger, deploymentManager, orderer.NewKahnRestoreLockOrderer(), executor.NewSerialExecutor())
}
