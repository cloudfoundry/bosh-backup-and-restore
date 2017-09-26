package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orderer"
)

func BuildDeploymentBackupChecker(target,
	username,
	password,
	caCert string,
	withDebug,
	withManifest bool) (*orchestrator.BackupChecker, error) {
	logger := BuildLogger(withDebug)

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

	return orchestrator.NewBackupChecker(logger, deploymentManager, orderer.NewKahnBackupLockOrderer()), nil
}
