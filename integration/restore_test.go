package integration

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/pivotal-cf/pcf-backup-and-restore/testcluster"

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

	Context("when deployment has a single instance", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(
				mockbosh.VMsForDeployment(deploymentName).RedirectsToTask(14),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(14).RespondsWithVMsOutput([]string{}),
				mockbosh.TaskOutput(14).RespondsWithVMsOutput([]mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
					},
				}),
				mockbosh.StartSSHSession(deploymentName).SetSSHResponseCallback(func(username, key string) {
					instance1.CreateUser(username, key)
				}).RedirectsToTask(15),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(15).RespondsWith("{}"),
				mockbosh.TaskOutput(15).RespondsWith(fmt.Sprintf(`[{"status":"success",
				"ip":"%s",
				"host_public_key":"not-relevant",
				"index":0}]`, instance1.Address())),
				mockbosh.CleanupSSHSession(deploymentName).RedirectsToTask(16),
				mockbosh.Task(16).RespondsWithTaskContainingState(mockbosh.TaskDone),
			)

			instance1.ScriptExist("/var/vcap/jobs/redis/bin/restore", `#!/usr/bin/env sh
touch /tmp/restored_file`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())

			file, err := os.Create(restoreWorkspace + "/" + deploymentName + "/" + "metadata")
			Expect(err).NotTo(HaveOccurred())

			_, err = file.Write([]byte(`---
instances:
- instance_name: redis-dedicated-node
  instance_id: 0
  checksum: foo
`))
			Expect(err).NotTo(HaveOccurred())
			Expect(file.Close()).To(Succeed())

			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_PASSWORD=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
			//instance1.Die()
		})

		It("does not fail", func() {
			Expect(session.ExitCode()).To(Equal(0))
		})

		It("restores from a local backup", func() {
			Expect(instance1.AssertFileExists("/tmp/restored_file")).To(BeTrue())
		})
	})
})
