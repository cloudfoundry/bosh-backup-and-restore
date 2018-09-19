package factory

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	boshcmd "github.com/cloudfoundry/bosh-cli/cmd"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

func BuildBoshClient(targetUrl, username, password, caCertPathOrValue string, logger boshlog.Logger) (bosh.Client, error) {
	var boshClient bosh.Client
	var err error
	fs := boshsys.NewOsFileSystem(logger)

	caCertArg := boshcmd.CACertArg{FS: fs}

	err = caCertArg.UnmarshalFlag(caCertPathOrValue)
	if err != nil {
		return boshClient, err
	}

	boshClient, err = bosh.BuildClient(targetUrl, username, password, caCertArg.Content, logger)
	if err != nil {
		return boshClient, err
	}

	return boshClient, nil
}
