package bosh

import (
	"encoding/pem"
	"log"
	"net/http/httptest"

	"github.com/cloudfoundry/bosh-backup-and-restore/internal/cf-webmock/mockbosh"
	"github.com/cloudfoundry/bosh-backup-and-restore/internal/cf-webmock/mockhttp"
	"github.com/cloudfoundry/bosh-backup-and-restore/internal/cf-webmock/mockuaa"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("BuildClient", func() {
	var logger = boshlog.New(boshlog.LevelDebug, log.New(gbytes.NewBuffer(), "[bosh-package] ", log.Lshortfile))

	var (
		director       *mockhttp.Server
		deploymentName = "my-little-deployment"
		bbrVersion     = "bbr_version"
		caCert         string
	)

	BeforeEach(func() {
		director = mockbosh.NewTLS()

		x509Cert := httptest.NewTLSServer(nil).Certificate()
		pem := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: x509Cert.Raw,
		})
		caCert = string(pem)
	})

	AfterEach(func() {
		director.VerifyMocks()
	})

	Context("With Basic Auth", func() {
		It("build the client which makes basic auth against director", func() {
			username := "foo"
			password := "bar"

			director.ExpectedBasicAuth(username, password)
			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeBasic(),
				mockbosh.Manifest(deploymentName).RespondsWith([]byte("manifest contents")),
			)

			client, err := BuildClient(director.URL, username, password, caCert, bbrVersion, logger)

			Expect(err).NotTo(HaveOccurred())
			manifest, err := client.GetManifest(deploymentName)
			Expect(err).NotTo(HaveOccurred())
			Expect(manifest).To(Equal("manifest contents"))
		})
	})

	Context("With UAA", func() {
		var uaaServer *mockuaa.ClientCredentialsServer

		It("build the client which makes basic auth against director", func() {
			username := "foo"
			password := "bar"
			uaaToken := "baz"

			uaaServer = mockuaa.NewClientCredentialsServerTLS(username, password, uaaToken)

			director.ExpectedAuthorizationHeader("bearer " + uaaToken)
			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeUAA(uaaServer.URL),
				mockbosh.Manifest(deploymentName).RespondsWith([]byte("manifest contents")),
			)

			client, err := BuildClient(director.URL, username, password, caCert, bbrVersion, logger)

			Expect(err).NotTo(HaveOccurred())
			manifest, err := client.GetManifest(deploymentName)
			Expect(err).NotTo(HaveOccurred())
			Expect(manifest).To(Equal("manifest contents"))
		})

		It("fails if uaa url is not valid", func() {
			username := "no-relevant"
			password := "no-relevant"

			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeUAA(""),
			)
			_, err := BuildClient(director.URL, username, password, caCert, bbrVersion, logger)

			Expect(err).To(MatchError(ContainSubstring("invalid UAA URL")))

		})
	})

	It("fails if CA cert value is invalid", func() {
		username := "no-relevant"
		password := "no-relevant"
		caCertPath := "-----BEGIN"
		basicAuthDirectorURL := director.URL

		_, err := BuildClient(basicAuthDirectorURL, username, password, caCertPath, bbrVersion, logger)
		Expect(err).To(MatchError(ContainSubstring("Missing PEM block")))
	})

	It("fails if invalid bosh url", func() {
		username := "no-relevant"
		password := "no-relevant"
		caCertPath := ""
		basicAuthDirectorURL := ""

		_, err := BuildClient(basicAuthDirectorURL, username, password, caCertPath, bbrVersion, logger)
		Expect(err).To(MatchError(ContainSubstring("invalid bosh URL")))
	})

	It("fails if info cant be retrieved", func() {
		username := "no-relevant"
		password := "no-relevant"

		director.VerifyAndMock(
			mockbosh.Info().Fails("fooo!"),
		)

		_, err := BuildClient(director.URL, username, password, caCert, bbrVersion, logger)
		Expect(err).To(MatchError(ContainSubstring("bosh director unreachable or unhealthy")))
	})
})
