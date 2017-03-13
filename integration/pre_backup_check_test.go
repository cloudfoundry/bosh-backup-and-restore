package integration

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/pivotal-cf/bosh-backup-and-restore/testcluster"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("Pre-backup checks", func() {
	var director *mockhttp.Server
	var backupWorkspace string
	var session *gexec.Session
	var deploymentName string
	var instance1 *testcluster.Instance

	BeforeEach(func() {
		deploymentName = "my-little-deployment"
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
		director.VerifyMocks()
	})

	JustBeforeEach(func() {
		session = runBinary(
			backupWorkspace,
			[]string{"BOSH_CLIENT_SECRET=admin"},
			"--ca-cert", sslCertPath,
			"--username", "admin",
			"--target", director.URL,
			"--deployment", deploymentName,
			"pre-backup-check",
		)
	})

	Context("When there is a deployment which has one instance", func() {
		singleInstanceResponse := func(instanceGroupName string) []mockbosh.VMsOutput {
			return []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: instanceGroupName,
				},
			}
		}

		Context("and there is a backup script", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				By("creating a dummy backup script")

				mockDirectorWith(director,
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					ManifestIsNotDownloaded(),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/b-backup", `#!/usr/bin/env sh
set -u
printf "backupcontent1" > $ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $ARTIFACT_DIRECTORY/backupdump2
`)

			})

			It("exits zero", func() {
				Expect(session.ExitCode()).To(BeZero())
			})

			It("outputs a log message saying the deployment can be backed up", func() {
				Expect(string(session.Out.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' can be backed up."))
			})
		})

		Context("if there are no backup scripts", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				mockDirectorWith(director,
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					ManifestIsNotDownloaded(),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

				instance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/not-a-backup-script",
				)
			})

			It("returns exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			It("prints an error", func() {
				Expect(string(session.Out.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' cannot be backed up."))
				Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' has no backup scripts"))
			})

			It("does not create a backup on disk", func() {
				Expect(path.Join(backupWorkspace, deploymentName)).NotTo(BeADirectory())
			})
		})
	})

	Context("When deployment does not exist", func() {
		BeforeEach(func() {
			deploymentName = "my-non-existent-deployment"
			director.VerifyAndMock(mockbosh.VMsForDeployment(deploymentName).NotFound())
		})

		It("returns exit code 1", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})

		It("prints an error", func() {
			Expect(string(session.Out.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' cannot be backed up."))
			Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
		})

	})

	Context("When the director is unreachable", func() {
		BeforeEach(func() {
			deploymentName = "my-director-is-broken"
			director.VerifyAndMock(VmsForDeploymentFails(deploymentName)...)
		})

		It("returns exit code 1", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})

		It("prints an error", func() {
			Expect(string(session.Out.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' cannot be backed up."))
			Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
		})
	})
})
