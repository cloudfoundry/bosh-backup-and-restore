package deployment

import (
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pre-backup checks", func() {
	var director *mockhttp.Server
	var backupWorkspace string
	var session *gexec.Session
	var deploymentName string
	manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
  - name: redis-writer
    release: redis
  - name: redis-broker
    release: redis
`

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

	Context("When run with --deployment flag", func() {
		JustBeforeEach(func() {
			session = binary.Run(
				backupWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", deploymentName,
				"pre-backup-check",
			)
		})

		Context("and there is a deployment which has one instance", func() {
			var instance1 *testcluster.Instance

			singleInstanceResponse := func(instanceGroupName string) []mockbosh.VMsOutput {
				return []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: instanceGroupName,
						ID:      "fake-uuid",
						Index:   newIndex(0),
					},
				}
			}

			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
			})

			AfterEach(func() {
				instance1.DieInBackground()
			})

			Context("and there is a backup script", func() {
				BeforeEach(func() {
					By("creating a dummy backup script")

					MockDirectorWith(director,
						mockbosh.Info().WithAuthTypeBasic(),
						VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
						DownloadManifest(deploymentName, manifest),
						SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
						CleanupSSH(deploymentName, "redis-dedicated-node"),
					)

					makeBackupable(instance1)

				})

				It("exits zero", func() {
					Expect(session.ExitCode()).To(BeZero())
				})

				It("outputs a log message saying the deployment can be backed up", func() {
					Expect(session.Out).To(gbytes.Say("Deployment '" + deploymentName + "' can be backed up."))
				})

				Context("but the pre-backup-lock ordering is cyclic", func() {
					BeforeEach(func() {
						instance1.CreateScript(
							"/var/vcap/jobs/redis/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/redis-pre-backup-lock-called
exit 0`)
						instance1.CreateScript(
							"/var/vcap/jobs/redis-writer/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/redis-writer-pre-backup-lock-called
exit 0`)
						instance1.CreateScript("/var/vcap/jobs/redis-writer/bin/bbr/metadata",
							`#!/usr/bin/env sh
echo "---
backup_should_be_locked_before:
- job_name: redis
  release: redis
"`)
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata",
							`#!/usr/bin/env sh
echo "---
backup_should_be_locked_before:
- job_name: redis-writer
  release: redis
"`)
					})

					It("Should fail", func() {
						By("exiting with an error", func() {
							Expect(session).To(gexec.Exit(1))
						})

						By("printing a helpful error message", func() {
							Expect(session.Err).To(gbytes.Say("job locking dependency graph is cyclic"))
						})
					})
				})

				Context("but the backup artifact directory already exists", func() {
					BeforeEach(func() {
						instance1.CreateDir("/var/vcap/store/bbr-backup")
					})

					It("returns exit code 1", func() {
						Expect(session.ExitCode()).To(Equal(1))
					})

					It("prints an error with a backup-cleanup footer", func() {
						Expect(session.Out).To(gbytes.Say("Deployment '" + deploymentName + "' cannot be backed up."))
						Expect(session.Err).To(gbytes.Say("Directory /var/vcap/store/bbr-backup already exists on instance redis-dedicated-node/fake-uuid"))
						Eventually(session.Err).Should(gbytes.Say("It is recommended that you run `bbr backup-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."))
						Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
					})
				})
			})

			Context("if there are no backup scripts", func() {
				BeforeEach(func() {
					MockDirectorWith(director,
						mockbosh.Info().WithAuthTypeBasic(),
						VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
						DownloadManifest(deploymentName, manifest),
						SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
						CleanupSSH(deploymentName, "redis-dedicated-node"),
					)

					instance1.CreateExecutableFiles(
						"/var/vcap/jobs/redis/bin/not-a-backup-script",
					)
				})

				It("returns exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("prints an error", func() {
					Expect(session.Out).To(gbytes.Say("Deployment '" + deploymentName + "' cannot be backed up."))
					Expect(session.Err).To(gbytes.Say("Deployment '" + deploymentName + "' has no backup scripts"))
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
				})

				It("writes the stack trace", func() {
					files, err := filepath.Glob(filepath.Join(backupWorkspace, "bbr-*.err.log"))
					Expect(err).NotTo(HaveOccurred())
					logFilePath := files[0]
					_, err = os.Stat(logFilePath)
					Expect(os.IsNotExist(err)).To(BeFalse())
					stackTrace, err := ioutil.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})

			})
		})

		Context("and the deployment does not exist", func() {
			BeforeEach(func() {
				deploymentName = "my-non-existent-deployment"
				director.VerifyAndMock(
					mockbosh.Info().WithAuthTypeBasic(),
					mockbosh.VMsForDeployment(deploymentName).NotFound(),
				)
			})

			It("returns exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			It("prints an error", func() {
				Expect(session.Out).To(gbytes.Say("Deployment '" + deploymentName + "' cannot be backed up."))
				Expect(session.Err).To(gbytes.Say("Director responded with non-successful status code"))
			})
		})

		Context("and the director is unreachable", func() {
			BeforeEach(func() {
				deploymentName = "my-director-is-broken"
				director.VerifyAndMock(
					AppendBuilders(
						InfoWithBasicAuth(),
						VmsForDeploymentFails(deploymentName),
					)...,
				)
			})

			It("returns exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			It("prints an error", func() {
				Expect(session.Out).To(gbytes.Say("Deployment '" + deploymentName + "' cannot be backed up."))
				Expect(session.Err).To(gbytes.Say("Director responded with non-successful status code"))
			})
		})
	})

	Context("When run with the --all-deployments flag", func() {
		JustBeforeEach(func() {
			session = binary.Run(
				backupWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--all-deployments",
				"pre-backup-check",
			)
		})

		var instance *testcluster.Instance
		var instance2 *testcluster.Instance

		singleInstanceResponse := func(instanceGroupName string) []mockbosh.VMsOutput {
			return []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: instanceGroupName,
					ID:      "fake-uuid",
					Index:   newIndex(0),
				},
			}
		}

		Context("and all deployments have backup scripts", func() {
			deploymentName1 := "deployment1"
			deploymentName2 := "deployment2"

			BeforeEach(func() {
				instance = testcluster.NewInstance()

				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deploymentName1, deploymentName2}),
					VmsForDeployment(deploymentName1, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName1, manifest),
					SetupSSH(deploymentName1, "redis-dedicated-node", "fake-uuid", 0, instance),
					CleanupSSH(deploymentName1, "redis-dedicated-node"),

					VmsForDeployment(deploymentName2, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName2, manifest),
					SetupSSH(deploymentName2, "redis-dedicated-node", "fake-uuid", 0, instance),
					CleanupSSH(deploymentName2, "redis-dedicated-node"),
				)...)

				makeBackupable(instance)
			})

			AfterEach(func() {
				instance.DieInBackground()
			})

			It("exits zero", func() {
				Expect(session.ExitCode()).To(BeZero())
			})

			It("outputs a log message saying the deployments can be backed up", func() {
				Expect(session.Out).To(gbytes.Say("Deployment '" + deploymentName1 + "' can be backed up."))
				Expect(session.Out).To(gbytes.Say("Deployment '" + deploymentName2 + "' can be backed up."))
				Expect(session.Out).To(gbytes.Say("All 2 deployments can be backed up"))
			})
		})

		Context("and there are no deployments", func() {
			BeforeEach(func() {
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{}),
				)...)

			})

			It("fails", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(session.Err).To(gbytes.Say("Failed to find any deployments"))
			})
		})

		Context("and fails to get deployments", func() {
			BeforeEach(func() {
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					DeploymentsFails("oups"),
				)...)

			})

			It("exits non-zero", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("prints an error", func() {
				Expect(session.Err).To(gbytes.Say("oups"))
			})
		})

		Context("and the backup directory already exists on one of the deployments", func() {
			deploymentName1 := "deployment1"
			deploymentName2 := "deployment2"

			BeforeEach(func() {
				instance = testcluster.NewInstance()
				instance2 = testcluster.NewInstance()

				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deploymentName1, deploymentName2}),
					VmsForDeployment(deploymentName1, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName1, manifest),
					SetupSSH(deploymentName1, "redis-dedicated-node", "fake-uuid", 0, instance),
					CleanupSSH(deploymentName1, "redis-dedicated-node"),

					VmsForDeployment(deploymentName2, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName2, manifest),
					SetupSSH(deploymentName2, "redis-dedicated-node", "fake-uuid", 0, instance2),
					CleanupSSH(deploymentName2, "redis-dedicated-node"),
				)...)

				makeBackupable(instance)
				makeBackupable(instance2)

				instance2.CreateDir("/var/vcap/store/bbr-backup")
			})

			AfterEach(func() {
				instance.DieInBackground()
				instance2.DieInBackground()
			})

			It("fails and outputs a log message saying which deployments can be backed up", func() {
				Expect(session.ExitCode()).To(Equal(1))

				Expect(session.Out).To(gbytes.Say("Deployment '" + deploymentName1 + "' can be backed up."))
				Expect(session.Out).To(gbytes.Say("Deployment '" + deploymentName2 + "' cannot be backed up."))
				Expect(session.Out).To(gbytes.Say("Directory /var/vcap/store/bbr-backup already exists on instance redis-dedicated-node/fake-uuid"))

				Expect(session.Err).To(gbytes.Say("1 out of 2 deployments cannot be backed up:\n%s", deploymentName2))
				Expect(session.Err).To(gbytes.Say("Deployment '%s':", deploymentName2))
				Expect(session.Err).To(gbytes.Say("Directory /var/vcap/store/bbr-backup already exists on instance redis-dedicated-node/fake-uuid"))
				Eventually(session.Err).Should(gbytes.Say("It is recommended that you run `bbr deployment --all-deployments backup-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."))
			})
		})
	})
})

func makeBackupable(instance *testcluster.Instance) {
	instance.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh
set -u
printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $BBR_ARTIFACT_DIRECTORY/backupdump2
`)
}
