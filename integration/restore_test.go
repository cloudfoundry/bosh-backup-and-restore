package integration

import (
	"io/ioutil"
	"os"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restore", func() {
	var director *mockhttp.Server
	var restoreWorkspace string

	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		restoreWorkspace, err = ioutil.TempDir(".", "restore-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(restoreWorkspace)).To(Succeed())
		director.VerifyMocks()
	})

	Context("when deployment is not present", func() {
		var session *gexec.Session

		BeforeEach(func() {
			director.VerifyAndMock(mockbosh.VMsForDeployment("my-new-deployment").NotFound())

			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_PASSWORD=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", "my-new-deployment",
				"restore")

		})
		It("fails", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})
		It("prints an error", func() {
			Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
		})
	})

	XContext("when deployment has a single instance", func() {
		var session *gexec.Session

		BeforeEach(func() {
			director.VerifyAndMock(
				mockbosh.VMsForDeployment("my-new-deployment").RedirectsToTask(14),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(14).RespondsWithVMsOutput([]string{}),
				mockbosh.TaskOutput(14).RespondsWithVMsOutput([]mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
					},
				}),
			)

			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_PASSWORD=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", "my-new-deployment",
				"restore")

		})

		It("does not fail", func() {
			Expect(session.ExitCode()).To(Equal(0))
		})
	})
})
