package deployment

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/onsi/gomega/gbytes"

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
		manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
`

		BeforeEach(func() {
			cleanupWorkspace, _ = ioutil.TempDir(".", "cleanup-workspace-")

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
		Context("with single deployment", func() {
			var session *gexec.Session
			var instance1 *testcluster.Instance
			var deployment1 = "dep1"
			manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
`

			BeforeEach(func() {
				cleanupWorkspace, _ = ioutil.TempDir(".", "cleanup-workspace-")

				instance1 = testcluster.NewInstance()
				director = mockbosh.NewTLS()
				director.ExpectedBasicAuth("admin", "admin")
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deployment1}),
					InfoWithBasicAuth(),
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

			It("successfully cleans up all deployments after a failed backup", func() {
				By("Removing the files", func() {
					Eventually(session.ExitCode()).Should(Equal(0))
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
					Expect(session.Out).To(gbytes.Say("Successfully cleaned up: %s", deployment1))
				})

				By("printing the backup progress to the screen", func() {
					logfilePath := fmt.Sprintf("%s_%s.log", deployment1, `(\d){8}T(\d){6}Z\b`)

					AssertOutputWithTimestamp(session.Out, []string{
						fmt.Sprintf("Pending: %s", deployment1),
						fmt.Sprintf("Starting cleanup of %s, log file: %s", deployment1, logfilePath),
						fmt.Sprintf("Finished cleanup of %s", deployment1),
						fmt.Sprintf("Successfully cleaned up: %s", deployment1),
					})
				})

				By("outputing the deployment logs to file", func() {
					files, err := filepath.Glob(filepath.Join(cleanupWorkspace, fmt.Sprintf("%s_*.log", deployment1)))
					Expect(err).NotTo(HaveOccurred())
					Expect(files).To(HaveLen(1))

					logFilePath := files[0]
					Expect(filepath.Base(logFilePath)).To(MatchRegexp(fmt.Sprintf("%s_%s.log", deployment1, `(\d){8}T(\d){6}Z\b`)))

					backupLogContent, err := ioutil.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())

					output := string(backupLogContent)

					Expect(output).To(ContainSubstring("INFO - Looking for scripts"))
					Expect(output).To(ContainSubstring("INFO - redis-dedicated-node/fake-uuid/redis/backup"))
					Expect(output).To(ContainSubstring("INFO - Running post-backup-unlock scripts..."))
					Expect(output).To(ContainSubstring("INFO - Finished running post-backup-unlock scripts."))
					Expect(output).To(ContainSubstring("INFO - 'dep1' cleaned up"))
				})

			})
		})

		Context("with no deployments", func() {
			var session *gexec.Session

			BeforeEach(func() {
				cleanupWorkspace, _ = ioutil.TempDir(".", "cleanup-workspace-")

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

			BeforeEach(func() {
				cleanupWorkspace, _ = ioutil.TempDir(".", "cleanup-workspace-")

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
			manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
`

			BeforeEach(func() {
				cleanupWorkspace, _ = ioutil.TempDir(".", "cleanup-workspace-")

				instance1 = testcluster.NewInstance()
				director = mockbosh.NewTLS()
				director.ExpectedBasicAuth("admin", "admin")
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deployment1}),
					InfoWithBasicAuth(),
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

			It("reports that the one deployment failed to clean up with the correct error message", func() {
				Eventually(session.ExitCode()).Should(Equal(1))
				Expect(session.Out).To(gbytes.Say("Failed to cleanup deployment '" + deployment1 + "'"))
				Expect(session.Out).To(gbytes.Say(fmt.Sprintf("ERROR: failed cleanup of %s", deployment1)))

				Expect(session.Out).To(gbytes.Say("INFO - Looking for scripts"))
				Expect(session.Out).To(gbytes.Say("INFO - redis-dedicated-node/fake-uuid/redis/post-backup-unlock"))
				Expect(session.Out).To(gbytes.Say("INFO - Running post-backup-unlock scripts..."))
				Expect(session.Out).To(gbytes.Say("INFO - Unlocking redis on redis-dedicated-node/fake-uuid..."))
				Expect(session.Out).To(gbytes.Say("ERROR - Error unlocking redis on redis-dedicated-node/fake-uuid."))
				Expect(session.Out).To(gbytes.Say("INFO - Finished running post-backup-unlock scripts."))

				Expect(session.Err).To(gbytes.Say("1 out of 1 deployments could not be cleaned up:\n  %s", deployment1))
				Expect(session.Err).To(gbytes.Say("Deployment '%s':", deployment1))
				Expect(session.Err).To(gbytes.Say("exit code 1"))
			})

			It("logs the output to file", func() {
				files, err := filepath.Glob(filepath.Join(cleanupWorkspace, fmt.Sprintf("%s_*.log", deployment1)))
				Expect(err).NotTo(HaveOccurred())
				Expect(files).To(HaveLen(1))

				logFilePath := files[0]
				Expect(filepath.Base(logFilePath)).To(MatchRegexp(fmt.Sprintf("%s_%s.log", deployment1, `(\d){8}T(\d){6}Z\b`)))

				backupLogContent, err := ioutil.ReadFile(logFilePath)
				Expect(err).ToNot(HaveOccurred())

				logContent := string(backupLogContent)

				Expect(logContent).To(ContainSubstring("INFO - Looking for scripts"))
				Expect(logContent).To(ContainSubstring("INFO - redis-dedicated-node/fake-uuid/redis/post-backup-unlock"))
				Expect(logContent).To(ContainSubstring("INFO - Running post-backup-unlock scripts..."))
				Expect(logContent).To(ContainSubstring("INFO - Unlocking redis on redis-dedicated-node/fake-uuid..."))
				Expect(logContent).To(ContainSubstring("ERROR - Error unlocking redis on redis-dedicated-node/fake-uuid."))
				Expect(logContent).To(ContainSubstring("INFO - Finished running post-backup-unlock scripts."))
			})
		})
	})
})
