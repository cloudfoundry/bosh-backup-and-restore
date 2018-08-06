package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	boshcmd "github.com/cloudfoundry/bosh-cli/cmd"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

func BuildBoshDeploymentManager(targetUrl, username, password, caCertPathOrValue string, logger boshlog.Logger, downloadManifest bool) (orchestrator.DeploymentManager, error) {
	fs := boshsys.NewOsFileSystem(logger)

	caCertArg := boshcmd.CACertArg{FS: fs}

	err := caCertArg.UnmarshalFlag(caCertPathOrValue)
	if err != nil {
		return nil, err
	}

	boshClient, err := bosh.BuildClient(targetUrl, username, password, caCertArg.Content, logger)
	if err != nil {
		return nil, err
	}

	return bosh.NewDeploymentManager(boshClient, logger, downloadManifest), nil
}
