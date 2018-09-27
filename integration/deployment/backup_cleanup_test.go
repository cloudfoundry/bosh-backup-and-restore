package deployment

import (
	"io/ioutil"
	"os"

	"github.com/onsi/gomega/gbytes"

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

	FContext("when running with --all-deployments", func() {
		Context("with multiple deployments", func() {
			var session *gexec.Session
			var instance1 *testcluster.Instance
			var instance2 *testcluster.Instance
			var deployment1 = "dep1"
			var deployment2 = "dep2"
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
				instance2 = testcluster.NewInstance()
				director = mockbosh.NewTLS()
				director.ExpectedBasicAuth("admin", "admin")
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deployment1, deployment2}),
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
					VmsForDeployment(deployment2, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
							ID:      "fake-uuid",
							Index:   newIndex(0),
						}}),
					DownloadManifest(deployment2, manifest),
					SetupSSH(deployment2, "redis-dedicated-node", "fake-uuid", 0, instance2),
					CleanupSSH(deployment2, "redis-dedicated-node"),
				)...)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", ``)
				instance1.CreateDir("/var/vcap/store/bbr-backup")

				instance2.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", ``)
				instance2.CreateDir("/var/vcap/store/bbr-backup")
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
				instance2.DieInBackground()
				Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
			})

			It("successfully cleans up all deployments after a failed backup", func() {
				Eventually(session.ExitCode()).Should(Equal(0))
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
				Expect(instance2.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
				Expect(session.Out).To(gbytes.Say("All 2 deployments were cleaned up"))
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
			var instance2 *testcluster.Instance
			var deployment1 = "dep1"
			var deployment2 = "dep2"
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
				instance2 = testcluster.NewInstance()
				director = mockbosh.NewTLS()
				director.ExpectedBasicAuth("admin", "admin")
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deployment1, deployment2}),
					VmsForDeployment(deployment1, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
							ID:      "fake-uuid",
							Index:   newIndex(0),
						}}),
					DownloadManifest(deployment1, manifest),
					SetupSSH(deployment1, "redis-dedicated-node", "fake-uuid", 0, instance1),
					VmsForDeployment(deployment2, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
							ID:      "fake-uuid",
							Index:   newIndex(0),
						}}),
					DownloadManifest(deployment2, manifest),
					SetupSSH(deployment2, "redis-dedicated-node", "fake-uuid", 0, instance2),
					CleanupSSH(deployment2, "redis-dedicated-node"),
				)...)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `exit 1`)
				instance1.CreateDir("/var/vcap/store/bbr-backup")

				instance2.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", ``)
				instance2.CreateDir("/var/vcap/store/bbr-backup")
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
				instance2.DieInBackground()
				Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
			})

			It("reports that 1 deployment succeeds and the other fails", func() {
				Eventually(session.ExitCode()).Should(Equal(1))
				Expect(session.Out).To(gbytes.Say("Failed to cleanup deployment '" + deployment1 + "'"))
				Expect(session.Out).To(gbytes.Say("Cleaned up deployment '" + deployment2 + "'"))
				Expect(session.Err).To(gbytes.Say("1 out of 2 deployments could not be cleaned up:\n%s", deployment1))
				Expect(session.Err).To(gbytes.Say("Deployment '%s':", deployment1))
				Expect(session.Err).To(gbytes.Say("exit code 1"))
			})
		})
	})
})
