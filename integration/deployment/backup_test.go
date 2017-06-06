package deployment

import (
	"archive/tar"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/pivotal-cf/bosh-backup-and-restore/testcluster"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/bosh-backup-and-restore/integration"
)

var _ = Describe("Backup", func() {
	var director *mockhttp.Server
	var backupWorkspace string
	var session *gexec.Session
	var deploymentName string
	var downloadManifest bool
	var instance1 *testcluster.Instance

	BeforeEach(func() {
		deploymentName = "my-little-deployment"
		downloadManifest = false
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		instance1.DieInBackground()
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
		director.VerifyMocks()
	})

	JustBeforeEach(func() {
		params := []string{
			"deployment",
			"--ca-cert", sslCertPath,
			"--username", "admin",
			"--target", director.URL,
			"--deployment", deploymentName,
			"--debug",
			"backup"}

		if downloadManifest {
			params = append(params, "--with-manifest")
		}

		session = binary.Run(
			backupWorkspace,
			[]string{"BOSH_CLIENT_SECRET=admin"},
			params...,
		)
	})

	Context("When there is a deployment which has one instance", func() {
		var metadataFile string
		var redisNodeArtifactFile string

		singleInstanceResponse := func(instanceGroupName string) []mockbosh.VMsOutput {
			return []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: instanceGroupName,
				},
			}
		}

		Context("and there is a plausible backup script", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				By("creating a dummy backup script")
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh

set -u

printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $BBR_ARTIFACT_DIRECTORY/backupdump2
`)

				metadataFile = path.Join(backupWorkspace, deploymentName, "/metadata")
				redisNodeArtifactFile = path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0-redis.tar")
			})

			Context("and we don't ask for the manifest to be downloaded", func() {
				BeforeEach(func() {
					MockDirectorWith(director,
						mockbosh.Info().WithAuthTypeBasic(),
						VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
						SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
						CleanupSSH(deploymentName, "redis-dedicated-node"))
				})
				It("successfully backs up the deployment", func() {

					By("not running non-existent pre-backup scripts")

					By("exiting zero", func() {
						Expect(session.ExitCode()).To(BeZero())
					})

					By("creating a backup directory which contains a backup artifact and a metadata file", func() {
						Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
						Expect(redisNodeArtifactFile).To(BeARegularFile())
						Expect(metadataFile).To(BeARegularFile())
					})

					By("having successfully run the backup script, using the $BBR_ARTIFACT_DIRECTORY variable", func() {
						Expect(filesInTar(redisNodeArtifactFile)).To(ConsistOf("backupdump1", "backupdump2"))
						Expect(contentsInTar(redisNodeArtifactFile, "backupdump1")).To(Equal("backupcontent1"))
						Expect(contentsInTar(redisNodeArtifactFile, "backupdump2")).To(Equal("backupcontent2"))
					})

					By("correctly populating the metadata file", func() {
						metadataContents := ParseMetadata(metadataFile)

						currentTimezone, _ := time.Now().Zone()
						Expect(metadataContents.BackupActivityMetadata.StartTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))
						Expect(metadataContents.BackupActivityMetadata.FinishTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))

						Expect(metadataContents.InstancesMetadata).To(HaveLen(1))
						Expect(metadataContents.InstancesMetadata[0].InstanceName).To(Equal("redis-dedicated-node"))
						Expect(metadataContents.InstancesMetadata[0].InstanceIndex).To(Equal("0"))

						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Name).To(Equal("redis"))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums).To(HaveLen(2))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./redis/backupdump1"]).To(Equal(shaFor("backupcontent1")))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./redis/backupdump2"]).To(Equal(shaFor("backupcontent2")))

						Expect(metadataContents.CustomArtifactsMetadata).To(BeEmpty())
					})

					By("printing the backup progress to the screen", func() {
						Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Running pre-checks for backup of %s...", deploymentName)))
						Expect(session.Out).To(gbytes.Say("INFO - Scripts found:"))
						Expect(session.Out).To(gbytes.Say("INFO - redis-dedicated-node/fake-uuid/redis/backup"))
						Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Starting backup of %s...", deploymentName)))
						Expect(session.Out).To(gbytes.Say("INFO - Running pre-backup scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Done."))
						Expect(session.Out).To(gbytes.Say("INFO - Running backup scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Backing up redis on redis-dedicated-node/fake-uuid..."))
						Expect(session.Out).To(gbytes.Say("INFO - Done."))
						Expect(session.Out).To(gbytes.Say("INFO - Running post-backup scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Done."))
						Expect(session.Out).To(gbytes.Say("INFO - Copying backup -- [^-]*-- from redis-dedicated-node/fake-uuid..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished copying backup -- from redis-dedicated-node/fake-uuid..."))
						Expect(session.Out).To(gbytes.Say("INFO - Starting validity checks"))
						Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./redis/backupdump[12]`))
						Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./redis/backupdump[12]`))
						Expect(session.Out).To(gbytes.Say("DEBUG - Calculating shasum for remote files"))
						Expect(session.Out).To(gbytes.Say("DEBUG - Comparing shasums"))
						Expect(session.Out).To(gbytes.Say("INFO - Finished validity checks"))
					})

					By("cleaning up backup artifacts from the remote", func() {
						Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
					})
				})

				Context("and there is a metadata script which produces yaml containing the custom backup_name", func() {
					var redisCustomArtifactFile string
					var redisDefaultArtifactFile string

					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
	touch /tmp/metadata-script-was-run
echo "---
backup_name: custom_backup_named_redis
"`)
						redisCustomArtifactFile = path.Join(backupWorkspace, deploymentName, "/custom_backup_named_redis.tar")
						redisDefaultArtifactFile = path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0-redis.tar")
					})
					It("creates a named artifact", func() {
						By("runs the metadata scripts", func() {
							Expect(instance1.FileExists("/tmp/metadata-script-was-run")).To(BeTrue())
						})

						By("creating a custom backup artifact", func() {
							Expect(filesInTar(redisCustomArtifactFile)).To(ConsistOf("backupdump1", "backupdump2"))
							Expect(contentsInTar(redisCustomArtifactFile, "backupdump1")).To(Equal("backupcontent1"))
							Expect(contentsInTar(redisCustomArtifactFile, "backupdump2")).To(Equal("backupcontent2"))
						})

						By("not creating an artifact with the default name", func() {
							Expect(redisDefaultArtifactFile).NotTo(BeARegularFile())
						})

						By("recording the artifact as a custom artifact in the backup metadata", func() {
							Expect(metadataFile).To(BeARegularFile())

							metadataContents := ParseMetadata(metadataFile)

							currentTimezone, _ := time.Now().Zone()
							Expect(metadataContents.BackupActivityMetadata.StartTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))
							Expect(metadataContents.BackupActivityMetadata.FinishTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))

							Expect(metadataContents.CustomArtifactsMetadata).To(HaveLen(1))
							Expect(metadataContents.CustomArtifactsMetadata[0].Name).To(Equal("custom_backup_named_redis"))
							Expect(metadataContents.CustomArtifactsMetadata[0].Checksums).To(HaveLen(2))
							Expect(metadataContents.CustomArtifactsMetadata[0].Checksums["./backupdump1"]).To(Equal(shaFor("backupcontent1")))
							Expect(metadataContents.CustomArtifactsMetadata[0].Checksums["./backupdump2"]).To(Equal(shaFor("backupcontent2")))
						})
					})
				})

				Context("and the pre-backup-lock script is present", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/pre-backup-lock-script-was-run
`)
						instance1.CreateScript("/var/vcap/jobs/redis-broker/bin/bbr/pre-backup-lock", ``)
					})

					It("executes and logs the locks", func() {
						By("running the pre-backup-lock script", func() {
							Expect(instance1.FileExists("/tmp/pre-backup-lock-script-was-run")).To(BeTrue())
						})

						By("logging that it is locking the instance, and listing the scripts", func() {
							assertOutput(session, []string{
								`Locking redis on redis-dedicated-node/fake-uuid for backup`,
								"> /var/vcap/jobs/redis/bin/bbr/pre-backup-lock",
								"> /var/vcap/jobs/redis-broker/bin/bbr/pre-backup-lock",
							})
						})
					})

				})

				Context("when the pre-backup-lock script fails", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
echo 'ultra-bar'
(>&2 echo 'ultra-baz')
touch /tmp/pre-backup-lock-output
exit 1
`)
						instance1.CreateScript("/var/vcap/jobs/redis-broker/bin/bbr/pre-backup-lock", ``)
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
touch /tmp/post-backup-unlock-output
`)
					})

					It("logs the failure, and unlocks the system", func() {
						By("runs the pre-backup-lock scripts", func() {
							Expect(instance1.FileExists("/tmp/pre-backup-lock-output")).To(BeTrue())
						})

						By("exits with the correct error code", func() {
							Expect(session.ExitCode()).To(Equal(4))
						})

						By("logs the error", func() {
							Expect(session.Err.Contents()).To(ContainSubstring("pre backup lock script for job redis failed on redis-dedicated-node/fake-uuid."))
						})

						By("logs stdout", func() {
							Expect(session.Err.Contents()).To(ContainSubstring("Stdout: ultra-bar"))
						})

						By("logs stderr", func() {
							Expect(session.Err.Contents()).To(ContainSubstring("Stderr: ultra-baz"))
						})

						By("also runs the post-backup-unlock scripts", func() {
							Expect(instance1.FileExists("/tmp/post-backup-unlock-output")).To(BeTrue())
						})
					})

				})

				Context("when backup file has owner only permissions of different user", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh

set -u

dd if=/dev/urandom of=$BBR_ARTIFACT_DIRECTORY/backupdump1 bs=1KB count=1024
dd if=/dev/urandom of=$BBR_ARTIFACT_DIRECTORY/backupdump2 bs=1KB count=1024

mkdir $BBR_ARTIFACT_DIRECTORY/backupdump3
dd if=/dev/urandom of=$BBR_ARTIFACT_DIRECTORY/backupdump3/dump bs=1KB count=1024

chown vcap:vcap $BBR_ARTIFACT_DIRECTORY/backupdump3
chmod 0700 $BBR_ARTIFACT_DIRECTORY/backupdump3`)
					})
					It("backup is still drained", func() {
						By("exits zero", func() {
							Expect(session.ExitCode()).To(BeZero())
						})

						By("prints the artifact size with the files from the other users", func() {
							Eventually(session).Should(gbytes.Say("Copying backup -- 3.0M uncompressed -- from redis-dedicated-node/fake-uuid..."))
						})
					})
				})

				Context("when deployment has a post-backup-unlock script", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
touch /tmp/post-backup-unlock-script-was-run
echo "Unlocking release"`)
					})

					It("prints unlock progress to the screen", func() {
						By("runs the pre-backup-lock scripts", func() {
							Expect(instance1.FileExists("/tmp/post-backup-unlock-script-was-run")).To(BeTrue())
						})

						By("logging the script action", func() {
							assertOutput(session, []string{
								"Running unlock on redis-dedicated-node/fake-uuid",
								"Done.",
							})
						})
					})

				})

				Context("when the post backup unlock script fails", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
echo 'ultra-bar'
(>&2 echo 'ultra-baz')
exit 1`)
					})

					It("exits and prints the error", func() {
						By("exits with the correct error code", func() {
							Expect(session).To(gexec.Exit(8))
						})

						By("prints stdout", func() {
							Expect(session.Err.Contents()).To(ContainSubstring("Stdout: ultra-bar"))
						})

						By("prints stderr", func() {
							Expect(session.Err.Contents()).To(ContainSubstring("Stderr: ultra-baz"))
						})

						By("prints an error", func() {
							Expect(session.Err.Contents()).To(ContainSubstring("unlock script for job redis failed on redis-dedicated-node/fake-uuid."))
						})
					})

				})
			})

			Context("and we ask for the manifest to be downloaded", func() {
				BeforeEach(func() {
					downloadManifest = true

					director.VerifyAndMock(AppendBuilders(
						[]mockhttp.MockedResponseBuilder{mockbosh.Info().WithAuthTypeBasic()},
						VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
						SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
						DownloadManifest(deploymentName, "this is a totally valid yaml"),
						CleanupSSH(deploymentName, "redis-dedicated-node"),
					)...)
				})

				It("downloads the manifest", func() {
					Expect(path.Join(backupWorkspace, deploymentName, "manifest.yml")).To(BeARegularFile())
					Expect(ioutil.ReadFile(path.Join(backupWorkspace, deploymentName, "manifest.yml"))).To(Equal([]byte("this is a totally valid yaml")))
				})
			})
		})

		Context("when a deployment can't be backed up", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

				instance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/ctl",
				)
			})
			It("exits and displays a message", func() {
				Expect(session.ExitCode()).NotTo(BeZero(), "returns a non-zero exit code")
				Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment '"+deploymentName+"' has no backup scripts"),
					"prints an error")
				Expect(path.Join(backupWorkspace, deploymentName)).NotTo(BeADirectory(), "does not create a backup on disk")
			})
		})

		Context("when the instance backup script fails", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/bbr/backup", "echo 'ultra-bar'; (>&2 echo 'ultra-baz'); exit 1",
				)
			})

			It("returns exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})
		})

		Context("when both the instance backup script and cleanup fail", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSHFails(deploymentName, "redis-dedicated-node", "ultra-foo"),
				)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/bbr/backup", "(>&2 echo 'ultra-baz'); exit 1",
				)
			})

			It("exits correctly and prints an error", func() {
				By("returning exit code 17 (16 + 1)", func() {
					Expect(session.ExitCode()).To(Equal(17))
				})

				By("printing an error", func() {
					assertErrorOutput(session, []string{
						"backup script for job redis failed on redis-dedicated-node/fake-uuid.",
						"ultra-baz",
						"ultra-foo",
					})
				})
			})

		})

		Context("when backup succeeds but cleanup fails", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSHFails(deploymentName, "redis-dedicated-node", "Can't do it mate"),
				)

				instance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)
			})

			It("exits correctly and prints the error", func() {
				By("returning the correct error code", func() {
					Expect(session.ExitCode()).To(Equal(16))
				})

				By("printing an error", func() {
					Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' failed while cleaning up with error: "))
				})

				By("including the failure message in error output", func() {
					Expect(string(session.Err.Contents())).To(ContainSubstring("Can't do it mate"))
				})

				By("creating a backup on disk", func() {
					Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
				})
			})

		})

		Context("when running the metadata script does not give valid yml", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
touch /tmp/metadata-script-was-run-but-produces-invalid-yaml
echo "not valid yaml
"`)

				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)
			})

			It("attempts to use the metadata, and exits with an error", func() {
				By("running the metadata scripts", func() {
					Expect(instance1.FileExists("/tmp/metadata-script-was-run-but-produces-invalid-yaml")).To(BeTrue())
				})

				By("exiting with the correct error code", func() {
					Expect(session).To(gexec.Exit(1))
				})
			})

		})

		Context("when the artifact exists locally", func() {
			BeforeEach(func() {
				director.VerifyAndMock(mockbosh.Info().WithAuthTypeBasic())
				deploymentName = "already-backed-up-deployment"
				err := os.Mkdir(path.Join(backupWorkspace, deploymentName), 0777)
				Expect(err).ToNot(HaveOccurred())
			})

			It("fails and prints an error", func() {
				By("returning a non-zero exit code", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				By("printing an error", func() {
					Expect(string(session.Err.Contents())).To(
						ContainSubstring(
							fmt.Sprintf("artifact %s already exists", deploymentName),
						),
					)
				})
			})

		})
	})

	Context("When there is a deployment which has two instances", func() {
		twoInstancesResponse := func(firstInstanceGroupName, secondInstanceGroupName string) []mockbosh.VMsOutput {

			return []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: firstInstanceGroupName,
				},
				{
					IPs:     []string{"10.0.0.2"},
					JobName: secondInstanceGroupName,
				},
			}
		}

		Context("one backupable", func() {
			var backupableInstance, nonBackupableInstance *testcluster.Instance

			BeforeEach(func() {
				deploymentName = "my-bigger-deployment"
				backupableInstance = testcluster.NewInstance()
				nonBackupableInstance = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, twoInstancesResponse("redis-dedicated-node", "redis-broker")),
					append(SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, backupableInstance),
						SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, nonBackupableInstance)...),
					append(CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-broker")...),
				)
				backupableInstance.CreateFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)
			})

			AfterEach(func() {
				backupableInstance.DieInBackground()
				nonBackupableInstance.DieInBackground()
			})

			It("backs up deployment successfully", func() {
				Expect(session.ExitCode()).To(BeZero())
				Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
				Expect(path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0-redis.tar")).To(BeARegularFile())
				Expect(path.Join(backupWorkspace, deploymentName, "/redis-broker-0-redis.tar")).ToNot(BeAnExistingFile())
			})
		})

		Context("both backupable", func() {
			var backupableInstance1, backupableInstance2 *testcluster.Instance

			BeforeEach(func() {
				deploymentName = "my-two-instance-deployment"
				backupableInstance1 = testcluster.NewInstance()
				backupableInstance2 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, twoInstancesResponse("redis-dedicated-node", "redis-broker")),
					append(SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, backupableInstance1),
						SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, backupableInstance2)...),
					append(CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-broker")...),
				)

				backupableInstance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)

				backupableInstance2.CreateFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)

			})

			AfterEach(func() {
				backupableInstance1.DieInBackground()
				backupableInstance2.DieInBackground()
			})

			It("backs up both instances and prints process to the screen", func() {
				By("backing up both instances successfully", func() {
					Expect(session.ExitCode()).To(BeZero())
					Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
					Expect(path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0-redis.tar")).To(BeARegularFile())
					Expect(path.Join(backupWorkspace, deploymentName, "/redis-broker-0-redis.tar")).To(BeARegularFile())
				})

				By("printing the backup progress to the screen", func() {
					assertOutput(session, []string{
						fmt.Sprintf("Starting backup of %s...", deploymentName),
						"Backing up redis on redis-dedicated-node/fake-uuid...",
						"Backing up redis on redis-broker/fake-uuid-2...",
						"Done.",
						"Copying backup --",
						"from redis-dedicated-node/fake-uuid...",
						"from redis-broker/fake-uuid-2...",
						"Done.",
						fmt.Sprintf("Backup created of %s on", deploymentName),
					})
				})
			})

			Context("and the backup artifact directory already exists on one of them", func() {
				BeforeEach(func() {
					backupableInstance2.CreateDir("/var/vcap/store/bbr-backup")
				})

				It("fails without destroying existing artifact", func() {
					By("failing", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					By("not deleting the existing backup artifact directory", func() {
						Expect(backupableInstance2.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
					})

					By("loging which instance has the extant artifact directory", func() {
						Expect(session.Err).To(gbytes.Say("Directory /var/vcap/store/bbr-backup already exists on instance redis-broker/fake-uuid-2"))
					})
				})
			})
		})

		Context("and both specify the same backup name in their metadata", func() {
			var backupableInstance1, backupableInstance2 *testcluster.Instance

			BeforeEach(func() {
				deploymentName = "my-two-instance-deployment"
				backupableInstance1 = testcluster.NewInstance()
				backupableInstance2 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, twoInstancesResponse("redis-dedicated-node", "redis-broker")),
					append(SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, backupableInstance1),
						SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, backupableInstance2)...),
					append(CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-broker")...),
				)

				backupableInstance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)

				backupableInstance2.CreateFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)

				backupableInstance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
backup_name: duplicate_name
"`)
				backupableInstance2.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
backup_name: duplicate_name
"`)
			})

			AfterEach(func() {
				backupableInstance1.DieInBackground()
				backupableInstance2.DieInBackground()
			})

			It("fails correctly, and doesn't create artifacts", func() {
				By("not creating a file with the duplicated backup name", func() {
					Expect(path.Join(backupWorkspace, deploymentName, "/duplicate_name.tar")).NotTo(BeARegularFile())
				})

				By("refusing to perform backup", func() {
					Expect(session.Err.Contents()).To(ContainSubstring(
						"Multiple jobs in deployment 'my-two-instance-deployment' specified the same backup name",
					))
				})

				By("returning exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

		})

		Context("and one instance consumes restore custom name, which no instance provides", func() {
			var restoreInstance, backupableInstance *testcluster.Instance

			BeforeEach(func() {
				deploymentName = "my-two-instance-deployment"
				restoreInstance = testcluster.NewInstance()
				backupableInstance = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, twoInstancesResponse("redis-dedicated-node", "redis-broker")),
					append(SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, restoreInstance),
						SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, backupableInstance)...),
					append(CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-broker")...),
				)

				restoreInstance.CreateFiles(
					"/var/vcap/jobs/redis/bin/bbr/restore",
				)

				backupableInstance.CreateFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)

				restoreInstance.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
restore_name: name_1
"`)
				backupableInstance.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
backup_name: name_2
"`)
			})

			AfterEach(func() {
				restoreInstance.DieInBackground()
				backupableInstance.DieInBackground()
			})

			It("doesn't perform a backup", func() {
				By("refusing to perform backup", func() {
					Expect(string(session.Err.Contents())).To(ContainSubstring(
						"The redis-dedicated-node restore script expects a backup script which produces name_1 artifact which is not present in the deployment",
					))
				})
				By("returning exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

		})
	})

	Context("When deployment does not exist", func() {
		BeforeEach(func() {
			deploymentName = "my-non-existent-deployment"
			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeBasic(),
				mockbosh.VMsForDeployment(deploymentName).NotFound(),
			)
		})

		It("errors and exits", func() {
			By("returning exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
			})
		})

	})
})

func getTarReader(path string) *tar.Reader {
	reader, err := os.Open(path)
	Expect(err).NotTo(HaveOccurred())
	tarReader := tar.NewReader(reader)
	return tarReader
}

func filesInTar(path string) []string {
	tarReader := getTarReader(path)

	filenames := []string{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		info := header.FileInfo()
		if !info.IsDir() {
			filenames = append(filenames, info.Name())
		}
	}
	return filenames
}

func contentsInTar(tarFile, file string) string {
	tarReader := getTarReader(tarFile)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		info := header.FileInfo()
		if !info.IsDir() && info.Name() == file {
			contents, err := ioutil.ReadAll(tarReader)
			Expect(err).NotTo(HaveOccurred())
			return string(contents)
		}
	}
	Fail("File " + file + " not found in tar " + tarFile)
	return ""
}

func shaFor(contents string) string {
	shasum := sha256.New()
	shasum.Write([]byte(contents))
	return fmt.Sprintf("%x", shasum.Sum(nil))
}

func assertOutput(session *gexec.Session, strings []string) {
	for _, str := range strings {
		Expect(string(session.Out.Contents())).To(ContainSubstring(str))
	}
}

func assertErrorOutput(session *gexec.Session, strings []string) {
	for _, str := range strings {
		Expect(string(session.Err.Contents())).To(ContainSubstring(str))
	}
}
