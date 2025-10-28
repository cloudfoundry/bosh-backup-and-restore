package factory

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/bosh"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	boshcmd "github.com/cloudfoundry/bosh-cli/v7/cmd/opts"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

func BuildBoshClient(targetUrl, username, password, caCertPathOrValue, bbrVersion string, logger boshlog.Logger) (bosh.Client, error) {
	var boshClient bosh.Client
	var err error
	fs := boshsys.NewOsFileSystem(logger)

	caCertArg := boshcmd.CACertArg{FS: fs}

	err = caCertArg.UnmarshalFlag(caCertPathOrValue)
	if err != nil {
		return boshClient, err
	}

	boshClient, err = bosh.BuildClient(targetUrl, username, password, caCertArg.Content, bbrVersion, logger)
	if err != nil {
		return boshClient, err
	}

	return boshClient, nil
}
