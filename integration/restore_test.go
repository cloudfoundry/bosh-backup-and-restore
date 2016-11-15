package integration

import (
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
			director.VerifyAndMock(AppendBuilders(VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: "redis-dedicated-node",
				}}),
				SetupSSH(deploymentName, "redis-dedicated-node", instance1),
				CleanupSSH(deploymentName, "redis-dedicated-node"))...)

			instance1.ScriptExist("/var/vcap/jobs/redis/bin/restore", `#!/usr/bin/env sh
cp /var/vcap/store/backup/* /var/vcap/store/redis-server`)
			instance1.CreateDir("/var/vcap/store/redis-server")

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- instance_name: redis-dedicated-node
  instance_id: 0
  checksum: foo
`))

			backupContents, err := ioutil.ReadFile("../fixtures/backup.tgz")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0.tgz", backupContents)

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
			instance1.Die()
		})

		It("does not fail", func() {
			Expect(session.ExitCode()).To(Equal(0))
		})

		It("Untars the archive file on the remote", func() {
			Expect(instance1.AssertFileExists("/var/vcap/store/backup/backup.tgz")).To(BeTrue())
			Expect(instance1.AssertFileExists("/var/vcap/store/backup/redis-backup")).To(BeTrue())
		})

		It("Runs the restore script on the remote", func() {
			Expect(instance1.AssertFileExists("/var/vcap/store/redis-server/redis-backup"))
		})
	})
})

func createFileWithContents(filePath string, contents []byte) {
	file, err := os.Create(filePath)
	Expect(err).NotTo(HaveOccurred())
	_, err = file.Write([]byte(contents))
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())
}
