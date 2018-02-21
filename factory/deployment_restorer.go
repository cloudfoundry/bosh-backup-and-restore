package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/jobexecutor"
)

func BuildDeploymentRestorer(target,
	username,
	password,
	caCert string,
	debug bool) (*orchestrator.Restorer, error) {

	logger := BuildLogger(debug)
	deploymentManager, err := BuildBoshDeploymentManager(
		target,
		username,
		password,
		caCert,
		logger,
		false,
	)

	if err != nil {
		return nil, err
	}

	return orchestrator.NewRestorer(backup.BackupDirectoryManager{}, logger, deploymentManager, orderer.NewKahnRestoreLockOrderer(), jobexecutor.NewSerialJobExecutor()), nil
}
