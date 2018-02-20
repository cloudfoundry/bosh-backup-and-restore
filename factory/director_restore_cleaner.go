package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/standalone"
)

func BuildDirectorRestoreCleaner(host,
	username,
	privateKeyPath string,
	hasDebug bool) *orchestrator.RestoreCleaner {

	logger := BuildLogger(hasDebug)

	deploymentManager := standalone.NewDeploymentManager(logger,
		host,
		username,
		privateKeyPath,
		instance.NewJobFinder(logger),
		ssh.NewSshRemoteRunner,
	)

	return orchestrator.NewRestoreCleaner(logger, deploymentManager, orderer.NewDirectorLockOrderer())
}
