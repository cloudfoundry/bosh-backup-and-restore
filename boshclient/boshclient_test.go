package boshclient_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/pivotal-cf/pcf-backup-and-restore/boshclient"
)

var _ = Describe("BOSH client", func() {
	var director *mockhttp.Server

	BeforeEach(func() {
		director = mockbosh.New()
	})
	It("checks a deployment exists", func() {
		director.ExpectedBasicAuth("admin", "admin")
		director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").RespondsWith([]byte(`---
name: my-new-deployment`)))

		client := boshclient.New(director.URL, "admin", "admin")
		Expect(client.CheckDeploymentExists("my-new-deployment")).To(BeTrue())
	})
	It("returns false if deployment not found", func() {
		director.ExpectedBasicAuth("admin", "admin")
		director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").NotFound())

		client := boshclient.New(director.URL, "admin", "admin")
		Expect(client.CheckDeploymentExists("my-new-deployment")).To(BeFalse())
	})

	It("fails if bosh url is invalid", func() {
		client := boshclient.New("foo,bar,baz%", "not-relevant", "not-relevant")
		_, err := client.CheckDeploymentExists("not-relevant")
		Expect(err).To(HaveOccurred())
	})

	It("fails if request cannot be made", func() {
		client := boshclient.New("invalid.domain.thieone", "not-relevant", "not-relevant")
		_, err := client.CheckDeploymentExists("not-relevant")
		Expect(err).To(HaveOccurred())
	})

	It("fails if can't login", func() {
		director.ExpectedBasicAuth("admin", "admin")
		director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").RespondsWithUnauthorized("{}"))

		client := boshclient.New(director.URL, "admin", "admin")
		_, err := client.CheckDeploymentExists("my-new-deployment")
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("Invalid Credentials")))
	})

	It("fails if there was a server error", func() {
		director.ExpectedBasicAuth("admin", "admin")
		director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").Fails("Cant process this"))

		client := boshclient.New(director.URL, "admin", "admin")
		_, err := client.CheckDeploymentExists("my-new-deployment")
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("Cant process this")))
	})

})
