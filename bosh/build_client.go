package bosh

import (
	"io/ioutil"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"

	boshuaa "github.com/cloudfoundry/bosh-cli/uaa"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func BuildClient(targetUrl, username, password, caCertFileName string, logger boshlog.Logger) (BoshClient, error) {
	config, err := director.NewConfigFromURL(targetUrl)
	if err != nil {
		return nil, errors.Errorf("Target director URL is malformed - %s", err.Error())
	}

	var cert string
	if caCertFileName != "" {
		certBytes, err := ioutil.ReadFile(caCertFileName)
		if err != nil {
			return nil, err
		}
		cert = string(certBytes)
	}

	config.CACert = cert

	directorFactory := director.NewFactory(logger)
	infoDirector, err := directorFactory.New(config, director.NewNoopTaskReporter(), director.NewNoopFileReporter())
	if err != nil {
		return nil, err
	}

	info, err := infoDirector.Info()
	if err != nil {
		return nil, err
	}

	if info.Auth.Type == "uaa" {
		uaa, err := buildUaa(info, username, password, cert, logger)
		if err != nil {
			return nil, err
		}

		config.TokenFunc = boshuaa.NewClientTokenSession(uaa).TokenFunc
	} else {
		config.Client = username
		config.ClientSecret = password
	}

	boshDirector, err := directorFactory.New(config, director.NewNoopTaskReporter(), director.NewNoopFileReporter())
	if err != nil {
		return nil, err
	}

	return NewClient(boshDirector, director.NewSSHOpts, ssh.NewConnection, logger, instance.NewJobFinder(logger)), nil
}

func buildUaa(info director.Info, username, password, cert string, logger boshlog.Logger) (boshuaa.UAA, error) {
	urlAsInterface := info.Auth.Options["url"]
	url, ok := urlAsInterface.(string)
	if !ok {
		return nil, errors.Errorf("Expected URL '%s' to be a string", urlAsInterface)
	}

	uaaConfig, err := boshuaa.NewConfigFromURL(url)
	if err != nil {
		return nil, err
	}

	uaaConfig.CACert = cert
	uaaConfig.Client = username
	uaaConfig.ClientSecret = password

	return boshuaa.NewFactory(logger).New(uaaConfig)
}
