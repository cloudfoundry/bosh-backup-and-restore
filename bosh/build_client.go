package bosh

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry/bosh-cli/v7/director"
	"github.com/pkg/errors"

	boshuaa "github.com/cloudfoundry/bosh-cli/v7/uaa"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func BuildClient(targetUrl, username, password, caCert, bbrVersion string, logger boshlog.Logger) (Client, error) {
	var client Client

	factoryConfig, err := director.NewConfigFromURL(targetUrl)
	if err != nil {
		return client, errors.Errorf("invalid bosh URL - %s", err.Error())
	}

	factoryConfig.CACert = caCert

	directorFactory := director.NewFactory(logger)

	info, err := getDirectorInfo(directorFactory, factoryConfig)
	if err != nil {
		return client, err
	}

	if info.Auth.Type == "uaa" {
		uaa, err := buildUaa(info, username, password, caCert, logger)
		if err != nil {
			return client, err
		}

		factoryConfig.TokenFunc = boshuaa.NewClientTokenSession(uaa).TokenFunc
	} else {
		factoryConfig.Client = username
		factoryConfig.ClientSecret = password
	}

	boshDirector, err := directorFactory.New(factoryConfig, director.NewNoopTaskReporter(), director.NewNoopFileReporter())
	if err != nil {
		return client, errors.Wrap(err, "error building bosh director client")
	}

	return NewClient(boshDirector, director.NewSSHOpts, ssh.NewSshRemoteRunner, logger, instance.NewJobFinder(bbrVersion, logger), NewBoshManifestQuerier), nil
}

func getDirectorInfo(directorFactory director.Factory, factoryConfig director.FactoryConfig) (director.Info, error) {
	infoDirector, err := directorFactory.New(factoryConfig, director.NewNoopTaskReporter(), director.NewNoopFileReporter())
	if err != nil {
		return director.Info{}, errors.Wrap(err, "error building bosh director client")
	}

	info, err := infoDirector.Info()
	if err != nil {
		return director.Info{}, errors.Wrap(err, "bosh director unreachable or unhealthy")
	}

	return info, nil
}

func buildUaa(info director.Info, username, password, cert string, logger boshlog.Logger) (boshuaa.UAA, error) {
	urlAsInterface := info.Auth.Options["url"]
	url, ok := urlAsInterface.(string)
	if !ok {
		return nil, errors.Errorf("Expected URL '%s' to be a string", urlAsInterface)
	}

	uaaConfig, err := boshuaa.NewConfigFromURL(url)
	if err != nil {
		return nil, errors.Wrap(err, "invalid UAA URL")
	}

	uaaConfig.CACert = cert
	uaaConfig.Client = username
	uaaConfig.ClientSecret = password

	return boshuaa.NewFactory(logger).New(uaaConfig)
}
