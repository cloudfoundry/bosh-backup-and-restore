package deployment

import (
	"fmt"
	"github.com/onsi/gomega/gbytes"
	"io/ioutil"
	"os"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/integration"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

var _ = Describe("Backup cleanup", func() {
	var cleanupWorkspace string
	var director *mockhttp.Server

	Context("when deployment has a single instance", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string
		var err error
		manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
`

		BeforeEach(func() {
			cleanupWorkspace, err = ioutil.TempDir(".", "cleanup-workspace-")

			instance1 = testcluster.NewInstance()

			deploymentName = "my-new-deployment"
			director = mockbosh.NewTLS()
			director.ExpectedBasicAuth("admin", "admin")
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
						ID:      "fake-uuid",
						Index:   newIndex(0),
					}}),
				DownloadManifest(deploymentName, manifest),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSH(deploymentName, "redis-dedicated-node"),
			)...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", ``)
			instance1.CreateDir("/var/vcap/store/bbr-backup")
		})

		JustBeforeEach(func() {
			session = binary.Run(
				cleanupWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"backup-cleanup",
			)
		})

		AfterEach(func() {
			instance1.DieInBackground()
			director.VerifyMocks()
			Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
		})

		It("successfully cleans up a deployment after a failed backup", func() {
			Eventually(session.ExitCode()).Should(Equal(0))
			Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
		})
	})

	Context("when running with --all-deployments", func() {
		Context("with multiple deployments", func() {
			var session *gexec.Session
			var instance1 *testcluster.Instance
			var deployment1 = "dep1"
			var err error
			manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
`

			BeforeEach(func() {
				cleanupWorkspace, err = ioutil.TempDir(".", "cleanup-workspace-")

				instance1 = testcluster.NewInstance()
				director = mockbosh.NewTLS()
				director.ExpectedBasicAuth("admin", "admin")
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deployment1}),
					VmsForDeployment(deployment1, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
							ID:      "fake-uuid",
							Index:   newIndex(0),
						}}),
					DownloadManifest(deployment1, manifest),
					SetupSSH(deployment1, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deployment1, "redis-dedicated-node"),
				)...)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", ``)
				instance1.CreateDir("/var/vcap/store/bbr-backup")
			})

			JustBeforeEach(func() {
				session = binary.Run(
					cleanupWorkspace,
					[]string{"BOSH_CLIENT_SECRET=admin"},
					"deployment",
					"--ca-cert", sslCertPath,
					"--username", "admin",
					"--debug",
					"--target", director.URL,
					"--all-deployments",
					"backup-cleanup",
				)
			})

			AfterEach(func() {
				director.VerifyMocks()
				instance1.DieInBackground()
				Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
			})

			FIt("successfully cleans up all deployments after a failed backup", func() {
				By("Removing the files", func() {
					Eventually(session.ExitCode()).Should(Equal(0))
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
					Expect(session.Out).To(gbytes.Say("Successfully cleaned up: %s", deployment1))
				})

				By("printing the backup progress to the screen", func() {
					AssertOutputWithTimestamp(session.Out, []string{
						fmt.Sprintf("Pending: %s", deployment1),
						fmt.Sprintf("Starting cleanup of %s, log file: %s.log", deployment1, deployment1),
						fmt.Sprintf("Finished cleanup of %s", deployment1),
						fmt.Sprintf("Successfully cleaned up: %s", deployment1),
					})
				})

				By("outputing the deployment logs to file", func() {
					//logFilePath := fmt.Sprintf("%s.log", deployment1)
					//_, err := os.Stat(logFilePath)
					//Expect(os.IsNotExist(err)).To(BeFalse())
					//backupLogContent, err := ioutil.ReadFile(logFilePath)
					//Expect(err).ToNot(HaveOccurred())
					//
					//output := string(backupLogContent)
					//
					//Expect(output).To(ContainSubstring("INFO - Looking for scripts"))
					//Expect(output).To(ContainSubstring("INFO - redis/fake-uuid/redis/backup"))
					//Expect(output).To(ContainSubstring(fmt.Sprintf("INFO - Running pre-checks for backup of %s...", deployment1)))
					//Expect(output).To(ContainSubstring(fmt.Sprintf("INFO - Starting backup of %s...", deployment1)))
					//Expect(output).To(ContainSubstring("INFO - Running pre-backup-lock scripts..."))
					//Expect(output).To(ContainSubstring("INFO - Finished running pre-backup-lock scripts."))
					//Expect(output).To(ContainSubstring("INFO - Running backup scripts..."))
					//Expect(output).To(ContainSubstring("INFO - Backing up redis on redis/fake-uuid..."))
					//Expect(output).To(ContainSubstring("INFO - Finished running backup scripts."))
					//Expect(output).To(ContainSubstring("INFO - Running post-backup-unlock scripts..."))
					//Expect(output).To(ContainSubstring("INFO - Finished running post-backup-unlock scripts."))
					//Expect(output).To(MatchRegexp("INFO - Copying backup -- [^-]*-- for job redis on redis/fake-uuid..."))
					//Expect(output).To(ContainSubstring("INFO - Finished copying backup -- for job redis on redis/fake-uuid..."))
					//Expect(output).To(ContainSubstring("INFO - Starting validity checks -- for job redis on redis/fake-uuid..."))
					//Expect(output).To(ContainSubstring("INFO - Finished validity checks -- for job redis on redis/fake-uuid..."))
				})

			})
		})

		Context("with no deployments", func() {
			var session *gexec.Session
			var err error

			BeforeEach(func() {
				cleanupWorkspace, err = ioutil.TempDir(".", "cleanup-workspace-")

				director = mockbosh.NewTLS()
				director.ExpectedBasicAuth("admin", "admin")
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{}),
				)...)
			})

			JustBeforeEach(func() {
				session = binary.Run(
					cleanupWorkspace,
					[]string{"BOSH_CLIENT_SECRET=admin"},
					"deployment",
					"--ca-cert", sslCertPath,
					"--username", "admin",
					"--debug",
					"--target", director.URL,
					"--all-deployments",
					"backup-cleanup",
				)
			})

			AfterEach(func() {
				director.VerifyMocks()
				Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
			})

			It("fails", func() {
				Eventually(session.ExitCode()).Should(Equal(1))
				Expect(session.Err).To(gbytes.Say("Failed to find any deployments"))
			})
		})

		Context("and an error occurs while getting deployments", func() {
			var session *gexec.Session
			var err error

			BeforeEach(func() {
				cleanupWorkspace, err = ioutil.TempDir(".", "cleanup-workspace-")

				director = mockbosh.NewTLS()
				director.ExpectedBasicAuth("admin", "admin")
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					DeploymentsFails("Oopsy"),
				)...)
			})

			JustBeforeEach(func() {
				session = binary.Run(
					cleanupWorkspace,
					[]string{"BOSH_CLIENT_SECRET=admin"},
					"deployment",
					"--ca-cert", sslCertPath,
					"--username", "admin",
					"--debug",
					"--target", director.URL,
					"--all-deployments",
					"backup-cleanup",
				)
			})

			AfterEach(func() {
				director.VerifyMocks()
				Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
			})

			It("fails", func() {
				Eventually(session.ExitCode()).Should(Equal(1))
				Expect(session.Err).To(gbytes.Say("Oopsy"))
			})
		})

		Context("and an error occurs while unlocking a deployment", func() {
			var session *gexec.Session
			var instance1 *testcluster.Instance
			var deployment1 = "dep1"
			var err error
			manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
`

			BeforeEach(func() {
				cleanupWorkspace, err = ioutil.TempDir(".", "cleanup-workspace-")

				instance1 = testcluster.NewInstance()
				director = mockbosh.NewTLS()
				director.ExpectedBasicAuth("admin", "admin")
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deployment1}),
					VmsForDeployment(deployment1, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
							ID:      "fake-uuid",
							Index:   newIndex(0),
						}}),
					DownloadManifest(deployment1, manifest),
					SetupSSH(deployment1, "redis-dedicated-node", "fake-uuid", 0, instance1),
				)...)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `exit 1`)
				instance1.CreateDir("/var/vcap/store/bbr-backup")
			})

			JustBeforeEach(func() {
				session = binary.Run(
					cleanupWorkspace,
					[]string{"BOSH_CLIENT_SECRET=admin"},
					"deployment",
					"--ca-cert", sslCertPath,
					"--username", "admin",
					"--debug",
					"--target", director.URL,
					"--all-deployments",
					"backup-cleanup",
				)
			})

			AfterEach(func() {
				director.VerifyMocks()
				instance1.DieInBackground()
				Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
			})

			It("reports that the one deployment failed to clean up", func() {
				Eventually(session.ExitCode()).Should(Equal(1))
				Expect(session.Out).To(gbytes.Say("Failed to cleanup deployment '" + deployment1 + "'"))
				Expect(session.Err).To(gbytes.Say("1 out of 1 deployments could not be cleaned up:\n  %s", deployment1))
				Expect(session.Err).To(gbytes.Say("Deployment '%s':", deployment1))
				Expect(session.Err).To(gbytes.Say("exit code 1"))
			})
		})
	})
})
