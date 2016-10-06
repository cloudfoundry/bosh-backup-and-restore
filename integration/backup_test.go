package integration

import (
	"fmt"
	"os/exec"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var director *mockhttp.Server

var _ = Describe("Backup", func() {
	var commandPath string

	runBinary := func(params ...string) *gexec.Session {
		command := exec.Command(commandPath, params...)
		fmt.Fprintf(GinkgoWriter, "Running command:: %v", params)
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit())

		return session
	}

	BeforeEach(func() {
		var err error
		commandPath, err = gexec.Build("github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr")
		Expect(err).NotTo(HaveOccurred())

	})
	AfterEach(func() {
		director.VerifyMocks()
	})

	It("backs up deployment successfully", func() {
		director = mockbosh.New()
		director.ExpectedBasicAuth("admin", "admin")
		director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").RespondsWith([]byte(`---
name: my-new-deployment`)))

		session := runBinary("-u", "admin", "-p", "admin", "-t", director.URL, "-d", "my-new-deployment", "backup")

		Expect(session.ExitCode()).To(BeZero())
	})

	It("returns error if deployment not found", func() {
		director = mockbosh.New()
		director.ExpectedBasicAuth("admin", "admin")
		director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").NotFound())

		session := runBinary("-u", "admin", "-p", "admin", "-t", director.URL, "-d", "my-new-deployment", "backup")

		Expect(session.ExitCode()).To(Equal(1))
		Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment 'my-new-deployment' not found"))
	})
})
