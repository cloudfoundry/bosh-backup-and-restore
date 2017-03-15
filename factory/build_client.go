package factory

import (
	"fmt"
	"io/ioutil"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/pivotal-cf/bosh-backup-and-restore/bosh"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"

	boshuaa "github.com/cloudfoundry/bosh-cli/uaa"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func BuildClient(targetUrl, username, password, caCert string, logger boshlog.Logger) (orchestrator.BoshClient, error) {
	config, err := director.NewConfigFromURL(targetUrl)
	if err != nil {
		return nil, fmt.Errorf("Target director URL is malformed - %s", err.Error())
	}

	if caCert != "" {
		cert, err := ioutil.ReadFile(caCert)
		if err != nil {
			return nil, err
		}
		config.CACert = string(cert)
	}

	factory := director.NewFactory(logger)
	infoDirector, err := factory.New(config, director.NewNoopTaskReporter(), director.NewNoopFileReporter())

	info, _ := infoDirector.Info()

	if info.Auth.Type == "uaa" {
		uaaURL := info.Auth.Options["url"]

		uaaURLStr, ok := uaaURL.(string)
		if !ok {
			return nil, fmt.Errorf("Expected URL '%s' to be a string", uaaURL)
		}

		uaaConfig, err := boshuaa.NewConfigFromURL(uaaURLStr)
		if err != nil {
			return nil, err
		}

		if caCert != "" {
			cert, err := ioutil.ReadFile(caCert)
			if err != nil {
				return nil, err
			}
			uaaConfig.CACert = string(cert)
		}

		uaaConfig.Client = username
		uaaConfig.ClientSecret = password

		uaa, _ := boshuaa.NewFactory(logger).New(uaaConfig)

		config.TokenFunc = boshuaa.NewClientTokenSession(uaa).TokenFunc
	} else {
		config.Client = username
		config.ClientSecret = password
	}

	boshDirector, err := factory.New(config, director.NewNoopTaskReporter(), director.NewNoopFileReporter())
	if err != nil {
		return nil, err
	}

	return bosh.NewClient(boshDirector, director.NewSSHOpts, ssh.ConnectionCreator, logger, instance.NewJobFinder(logger)), nil
}
