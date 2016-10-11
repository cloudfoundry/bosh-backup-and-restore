package integration

import (
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup", func() {

	var director *mockhttp.Server
	AfterEach(func() {
		director.VerifyMocks()
	})
	BeforeEach(func() {
		director = mockbosh.New()
		director.ExpectedBasicAuth("admin", "admin")
	})

	It("backs up deployment successfully", func() {
		director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").RespondsWith([]byte(`---
name: my-new-deployment`)))

		session := runBinary([]string{"BOSH_PASSWORD=admin"}, "-u", "admin", "-t", director.URL, "-d", "my-new-deployment", "backup")

		Expect(session.ExitCode()).To(BeZero())
	})

	It("returns error if deployment not found", func() {
		director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").NotFound())

		session := runBinary([]string{"BOSH_PASSWORD=admin"}, "-u", "admin", "-t", director.URL, "-d", "my-new-deployment", "backup")

		Expect(session.ExitCode()).To(Equal(1))
		Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment 'my-new-deployment' not found"))
	})
})
