package bosh_test

import (
	"log"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/bosh-backup-and-restore/bosh"
)

var _ = Describe("BuildClient", func() {
	logger := boshlog.New(boshlog.LevelDebug, log.New(gbytes.NewBuffer(), "[bosh-package] ", log.Lshortfile), log.New(gbytes.NewBuffer(), "[bosh-package] ", log.Lshortfile))

	It("builds a Client that authenticates with HTTP Basic Auth", func() {
		username := MustHaveEnv("BOSH_CLIENT")
		password := MustHaveEnv("BASIC_AUTH_BOSH_CLIENT_SECRET")
		caCertPath := MustHaveEnv("BASIC_AUTH_BOSH_CERT_PATH")
		basicAuthDirectorUrl := MustHaveEnv("BASIC_AUTH_BOSH_URL")

		client, err := bosh.BuildClient(basicAuthDirectorUrl, username, password, caCertPath, logger)
		Expect(err).NotTo(HaveOccurred())

		_, err = client.GetManifest("does-not-exist")
		Expect(err.Error()).To(ContainSubstring("Director responded with non-successful status code '404'"))
	})

	XIt("builds a Client that authenticates with UAA", func() {
		username := MustHaveEnv("BOSH_CLIENT")
		password := MustHaveEnv("UAA_BOSH_CLIENT_SECRET")
		caCertPath := MustHaveEnv("UAA_BOSH_CERT_PATH")
		basicAuthDirectorUrl := MustHaveEnv("UAA_BOSH_URL")

		client, err := bosh.BuildClient(basicAuthDirectorUrl, username, password, caCertPath, logger)
		Expect(err).NotTo(HaveOccurred())

		_, err = client.GetManifest("does-not-exist")
		Expect(err.Error()).To(ContainSubstring("Director responded with non-successful status code '404'"))
	})
})
