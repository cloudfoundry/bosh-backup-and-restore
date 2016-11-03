package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/pivotal-cf/pcf-backup-and-restore/testcluster"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup", func() {
	var director *mockhttp.Server
	var backupWorkspace string

	AfterEach(func() {
		director.VerifyMocks()
	})
	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
	})

	Context("with deployment, with one instance present", func() {
		var instance1 *testcluster.Instance

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
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
				mockbosh.StartSSHSession("my-new-deployment").SetSSHResponseCallback(func(username, key string) {
					instance1.CreateUser(username, key)
				}).RedirectsToTask(15),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(15).RespondsWith("{}"),
				mockbosh.TaskOutput(15).RespondsWith(fmt.Sprintf(`[{"status":"success",
				"ip":"%s",
				"host_public_key":"not-relevant",
				"index":0}]`, instance1.Address())),
				mockbosh.CleanupSSHSession("my-new-deployment").RedirectsToTask(16),
				mockbosh.Task(16).RespondsWithTaskContainingState(mockbosh.TaskDone),
			)
		})

		AfterEach(func() {
			go instance1.Die()
		})

		It("backs up deployment successfully", func() {
			instance1.FilesExist(
				"/var/vcap/jobs/redis/bin/backup",
			)

			session := runBinary(backupWorkspace, []string{"BOSH_PASSWORD=admin"}, "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "--debug", "backup")

			Expect(session.ExitCode()).To(BeZero())
			Expect(path.Join(backupWorkspace, "my-new-deployment")).To(BeADirectory())
			// Expect(path.Join(backupWorkspace, "my-new-deployment/redis-0.tgz")).To(BeARegularFile())
		})

		It("errors if a deployment cant be backuped", func() {
			instance1.FilesExist(
				"/var/vcap/jobs/redis/bin/ctl",
			)

			session := runBinary(backupWorkspace, []string{"BOSH_PASSWORD=admin"}, "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "--debug", "backup")
			Expect(session.ExitCode()).NotTo(BeZero())
			Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment 'my-new-deployment' has no backup scripts"))
		})
	})

	It("returns error if deployment not found", func() {
		director.VerifyAndMock(mockbosh.VMsForDeployment("my-new-deployment").NotFound())

		session := runBinary(backupWorkspace, []string{"BOSH_PASSWORD=admin"}, "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")

		Expect(session.ExitCode()).To(Equal(1))
		Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
	})
})
