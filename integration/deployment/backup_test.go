package deployment

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/integration"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockbosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockhttp"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Backup", func() {
	var (
		director              *mockhttp.Server
		backupWorkspace       string
		session               *gexec.Session
		stdin                 io.WriteCloser
		deploymentName        string
		downloadManifest      bool
		unsafeLockFreeBackup  bool
		waitForBackupToFinish bool
		artifactPath          string
		verifyMocks           bool
		instance1             *testcluster.Instance
		manifest              string
	)

	backupDirectory := func() string {
		matches := possibleBackupDirectories(deploymentName, backupWorkspace)

		Expect(matches).To(HaveLen(1), "backup directory not found")
		return path.Join(backupWorkspace, matches[0])
	}

	metadataFile := func() string {
		return path.Join(backupDirectory(), "metadata")
	}

	artifactFile := func(name string) string {
		return path.Join(backupDirectory(), name)
	}

	BeforeEach(func() {
		deploymentName = "my-little-deployment"
		downloadManifest = false
		waitForBackupToFinish = true
		unsafeLockFreeBackup = false
		verifyMocks = true
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = os.MkdirTemp(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
		artifactPath = ""
		manifest = `---
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
- name: redis-broker
  instances: 1
  jobs:
  - name: redis
    release: redis
  - name: redis-writer
    release: redis
  - name: redis-broker
    release: redis
`
	})

	AfterEach(func() {
		if verifyMocks {
			director.VerifyMocks()
		}
		director.Close()

		instance1.DieInBackground()
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
	})

	JustBeforeEach(func() {
		env := []string{"BOSH_CLIENT_SECRET=admin"}

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

		if artifactPath != "" {
			params = append(params, "--artifact-path", artifactPath)
		}

		if unsafeLockFreeBackup {
			params = append(params, "--unsafe-lock-free")
		}

		if waitForBackupToFinish {
			session = binary.Run(backupWorkspace, env, params...)
		} else {
			session, stdin = binary.Start(backupWorkspace, env, params...)
			Eventually(session).Should(gbytes.Say(".+"))
		}
	})

	Context("When there is a deployment which has one instance", func() {
		singleInstanceResponse := func(instanceGroupName string) []mockbosh.VMsOutput {
			return []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: instanceGroupName,
					Index:   newIndex(0),
					ID:      "fake-uuid",
				},
			}
		}

		Context("and there is a plausible backup script", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				By("creating a dummy backup script")
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh

set -u
touch /tmp/backup-script-was-run
printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $BBR_ARTIFACT_DIRECTORY/backupdump2
`)
			})

			Context("and the bbr process receives SIGINT while backing up", func() {
				BeforeEach(func() {
					waitForBackupToFinish = false

					MockDirectorWith(director,
						mockbosh.Info().WithAuthTypeBasic(),
						VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
						DownloadManifest(deploymentName, manifest),
						SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
						CleanupSSH(deploymentName, "redis-dedicated-node"))

					By("creating a backup script that takes a while")
					instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh

						set -u

						sleep 5

						printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
					`)
				})

				Context("and the user decides to cancel the backup", func() {
					BeforeEach(func() {
						verifyMocks = false
					})

					It("terminates", func() {
						Eventually(session, "30s").Should(gbytes.Say("Backing up"))
						session.Interrupt()

						By("printing a helpful message and waiting for user input", func() {
							Consistently(session.Exited).ShouldNot(BeClosed(), "bbr exited without user confirmation")
							Eventually(session).Should(gbytes.Say(`Stopping a backup can leave the system in bad state. Are you sure you want to cancel\? \[yes/no\]`))
							Expect(string(session.Out.Contents())).To(HaveSuffix("[yes/no]\n"))
						})

						fmt.Fprintln(stdin, "yes") //nolint:errcheck

						By("then exiting with a failure", func() {
							Eventually(session, 10).Should(gexec.Exit(1))
						})

						By("outputting a warning about cleanup", func() {
							Eventually(session).Should(gbytes.Say("It is recommended that you run `bbr backup-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."))
						})

						By("not creating an artifact tar from the interrupted backup script", func() {
							boshBackupFilePath := path.Join(backupDirectory(), "/redis-dedicated-node-0-redis.tar")
							Expect(boshBackupFilePath).NotTo(BeAnExistingFile())
						})
					})
				})

				Context("and the user decides to continue backup", func() {
					It("continues to run", func() {
						session.Interrupt()

						By("printing a helpful message and waiting for user input", func() {
							Consistently(session.Exited).ShouldNot(BeClosed(), "bbr exited without user confirmation")
							Eventually(session).Should(gbytes.Say(`Stopping a backup can leave the system in bad state. Are you sure you want to cancel\? \[yes/no\]`))
							Expect(string(session.Out.Contents())).To(HaveSuffix("[yes/no]\n"))
						})

						fmt.Fprintln(stdin, "no") //nolint:errcheck

						By("waiting for the backup to finish successfully", func() {
							Eventually(session, 20).Should(gexec.Exit(0))
						})

						By("still completing the backup", func() {
							archive := OpenTarArchive(artifactFile("redis-dedicated-node-0-redis.tar"))

							Expect(archive.Files()).To(ConsistOf("backupdump1"))
							Expect(archive.FileContents("backupdump1")).To(Equal("backupcontent1"))
						})
					})
				})
			})

			Context("and we don't ask for the manifest to be downloaded", func() {
				BeforeEach(func() {
					MockDirectorWith(director,
						mockbosh.Info().WithAuthTypeBasic(),
						VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
						DownloadManifest(deploymentName, manifest),
						SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
						CleanupSSH(deploymentName, "redis-dedicated-node"))
				})

				It("successfully backs up the deployment", func() {
					By("not running non-existent pre-backup scripts")

					By("exiting zero", func() {
						Expect(session.ExitCode()).To(BeZero())
					})

					var redisNodeArchivePath string

					By("creating a backup directory which contains a backup artifact and a metadata file", func() {
						redisNodeArchivePath = artifactFile("redis-dedicated-node-0-redis.tar")
						Expect(backupDirectory()).To(BeADirectory())
						Expect(redisNodeArchivePath).To(BeARegularFile())
						Expect(metadataFile()).To(BeARegularFile())
					})

					By("having successfully run the backup script, using the $BBR_ARTIFACT_DIRECTORY variable", func() {
						archive := OpenTarArchive(redisNodeArchivePath)

						Expect(archive.Files()).To(ConsistOf("backupdump1", "backupdump2"))
						Expect(archive.FileContents("backupdump1")).To(Equal("backupcontent1"))
						Expect(archive.FileContents("backupdump2")).To(Equal("backupcontent2"))
					})

					By("correctly populating the metadata file", func() {
						metadataContents := ParseMetadata(metadataFile())

						currentTimezone, _ := time.Now().Zone()
						Expect(metadataContents.BackupActivityMetadata.StartTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))
						Expect(metadataContents.BackupActivityMetadata.FinishTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))

						Expect(metadataContents.InstancesMetadata).To(HaveLen(1))
						Expect(metadataContents.InstancesMetadata[0].InstanceName).To(Equal("redis-dedicated-node"))
						Expect(metadataContents.InstancesMetadata[0].InstanceIndex).To(Equal("0"))

						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Name).To(Equal("redis"))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums).To(HaveLen(2))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./backupdump1"]).To(Equal(ShaFor("backupcontent1")))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./backupdump2"]).To(Equal(ShaFor("backupcontent2")))

						Expect(metadataContents.CustomArtifactsMetadata).To(BeEmpty())
					})

					By("printing the backup progress to the screen", func() {
						Expect(session.Out).To(gbytes.Say("INFO - Looking for scripts"))
						Expect(session.Out).To(gbytes.Say("INFO - redis-dedicated-node/fake-uuid/redis/backup"))
						Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Running pre-checks for backup of %s...", deploymentName)))
						Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Starting backup of %s...", deploymentName)))
						Expect(session.Out).To(gbytes.Say("INFO - Running pre-backup-lock scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished running pre-backup-lock scripts."))
						Expect(session.Out).To(gbytes.Say("INFO - Running backup scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Backing up redis on redis-dedicated-node/fake-uuid..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished running backup scripts."))
						Expect(session.Out).To(gbytes.Say("INFO - Running post-backup-unlock scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished running post-backup-unlock scripts."))
						Expect(session.Out).To(gbytes.Say("INFO - Copying backup -- [^-]*-- for job redis on redis-dedicated-node/fake-uuid..."))
						Expect(session.Out).To(gbytes.Say(`INFO - Copying backup for job redis on redis-dedicated-node/fake-uuid -- \d\d\d?% complete`))
						Expect(session.Out).To(gbytes.Say("INFO - Finished copying backup -- for job redis on redis-dedicated-node/fake-uuid..."))
						Expect(session.Out).To(gbytes.Say("INFO - Starting validity checks -- for job redis on redis-dedicated-node/fake-uuid..."))
						Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./backupdump[12]`))
						Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./backupdump[12]`))
						Expect(session.Out).To(gbytes.Say("DEBUG - Calculating shasum for remote files"))
						Expect(session.Out).To(gbytes.Say("DEBUG - Comparing shasums"))
						Expect(session.Out).To(gbytes.Say("INFO - Finished validity checks -- for job redis on redis-dedicated-node/fake-uuid..."))

						Expect(string(session.Out.Contents())).NotTo(ContainSubstring("Skipping disabled jobs:"))
					})

					By("cleaning up backup artifacts from the remote", func() {
						Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
					})
				})

				Context("and the operator specifies an artifact path", func() {
					Context("and the artifact path is an existing directory", func() {
						BeforeEach(func() {
							var err error
							artifactPath, err = os.MkdirTemp("", "artifact-path-")
							Expect(err).NotTo(HaveOccurred())
						})

						AfterEach(func() {
							Expect(os.RemoveAll(artifactPath)).To(Succeed())
						})

						backupDirectoryWithArtifactPath := func() string {
							matches := possibleBackupDirectories(deploymentName, artifactPath)

							Expect(matches).To(HaveLen(1), "backup directory not found")
							return path.Join(artifactPath, matches[0])
						}

						It("should succeed and put the artifact into the artifact path", func() {
							By("exiting with exit code zero", func() {
								Expect(session.ExitCode()).To(BeZero())
							})

							By("placing the backup in a subdirectory of the artifact path", func() {
								Expect(backupDirectoryWithArtifactPath()).To(BeADirectory())
								Expect(path.Join(backupDirectoryWithArtifactPath(), "redis-dedicated-node-0-redis.tar")).To(BeARegularFile())
								Expect(path.Join(backupDirectoryWithArtifactPath(), "metadata")).To(BeARegularFile())
							})
						})
					})

					Context("and the artifact path does not exist", func() {
						BeforeEach(func() {
							artifactPath = "/not/a/valid/path"
						})

						It("should fail with an artifact directory does not exist error", func() {
							Expect(session.ExitCode()).NotTo(BeZero())
							Expect(session.Err).To(gbytes.Say(fmt.Sprintf("%s: no such file or directory", artifactPath)))
						})
					})
				})

				Context("and an addon with bbr scripts is installed on the instance", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/an_addon_job/bin/bbr/backup", `#!/usr/bin/env sh

echo "hi"
`)
					})

					It("takes a backup successfully", func() {
						By("not failing", func() {
							Expect(session.ExitCode()).To(BeZero(), string(session.Err.Contents()))
						})

						By("creating an artifact file", func() {
							Expect(artifactFile("redis-dedicated-node-0-redis.tar")).To(BeARegularFile())
						})
					})
				})

				Context("and there is a metadata script which produces yaml containing the custom backup_name", func() {
					var redisDefaultArtifactFile string

					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
	touch /tmp/metadata-script-was-run
echo "---
backup_name: custom_backup_named_redis
restore_name: custom_backup_named_redis
"`)
					})

					JustBeforeEach(func() {
						redisDefaultArtifactFile = path.Join(backupDirectory(), "/redis-dedicated-node-0-redis.tar")
					})

					It("creates a named artifact", func() {
						By("runs the metadata scripts", func() {
							Expect(instance1.FileExists("/tmp/metadata-script-was-run")).To(BeTrue())
						})

						By("printing a warning message", func() {
							Expect(session.Out).To(gbytes.Say("WARN - discontinued metadata keys backup_name/restore_name found in job redis. bbr will not be able to restore this backup artifact."))
						})

						By("running a the backup script", func() {
							Expect(instance1.FileExists("/tmp/backup-script-was-run")).To(BeTrue())
						})

						By("creating a default backup artifact", func() {
							archive := OpenTarArchive(redisDefaultArtifactFile)

							Expect(archive.Files()).To(ConsistOf("backupdump1", "backupdump2"))
							Expect(archive.FileContents("backupdump1")).To(Equal("backupcontent1"))
							Expect(archive.FileContents("backupdump2")).To(Equal("backupcontent2"))
						})

						By("recording the artifact in the metadata", func() {
							metadataContents := ParseMetadata(metadataFile())

							currentTimezone, _ := time.Now().Zone()
							Expect(metadataContents.BackupActivityMetadata.StartTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))
							Expect(metadataContents.BackupActivityMetadata.FinishTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))

							Expect(metadataContents.InstancesMetadata).To(HaveLen(1))
							Expect(metadataContents.InstancesMetadata[0].InstanceName).To(Equal("redis-dedicated-node"))
							Expect(metadataContents.InstancesMetadata[0].InstanceIndex).To(Equal("0"))

							Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Name).To(Equal("redis"))
							Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums).To(HaveLen(2))
							Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./backupdump1"]).To(Equal(ShaFor("backupcontent1")))
							Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./backupdump2"]).To(Equal(ShaFor("backupcontent2")))

							Expect(metadataContents.CustomArtifactsMetadata).To(BeEmpty())
						})
					})
				})

				Context("and there is a job present named mysql-backup", func() {
					BeforeEach(func() {
						manifest = `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
  - name: mysql-backup
    release: cf-backup-and-restore
`
						instance1.CreateScript("/var/vcap/jobs/mysql-backup/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
backup_name: mysql-artifact
"`)
						instance1.CreateScript("/var/vcap/jobs/mysql-backup/bin/bbr/backup", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/mysql-backup
touch /tmp/mysql-backup-script-was-run`)
					})

					It("ignores the mysql-backup job scripts", func() {
						By("exiting zero", func() {
							Expect(session.ExitCode()).To(BeZero())
						})

						By("running the redis backup script", func() {
							Expect(instance1.FileExists("/tmp/backup-script-was-run")).To(BeTrue())
						})

						By("not running the mysql-backup backup script", func() {
							Expect(instance1.FileExists("/tmp/mysql-backup-script-was-run")).To(BeFalse())
						})

						By("not printing a warning message", func() {
							Expect(string(session.Out.Contents())).NotTo(ContainSubstring("discontinued metadata keys backup_name/restore_name found"))
						})
					})
				})

				Context("and there is a metadata script which uses BBR_VERSION environment variable", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
	echo "${BBR_VERSION}" > /tmp/metadata-script-was-run
echo "---
"`)
					})

					It("calls the metadata script and passes the environment variable", func() {
						Expect(instance1.FileExists("/tmp/metadata-script-was-run")).To(BeTrue())
						Expect(strings.TrimSpace(instance1.GetFileContents("/tmp/metadata-script-was-run"))).To(Equal(bbrVersion))
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
							assertOutput(session.Out, []string{
								"> /var/vcap/jobs/redis-broker/bin/bbr/pre-backup-lock",
								"> /var/vcap/jobs/redis/bin/bbr/pre-backup-lock",
								`Locking redis on redis-dedicated-node/fake-uuid for backup`,
							})
						})
					})
				})
				Context("and the pre-backup-lock is present, but the unsafe-lock-free-backup is set", func() {
					BeforeEach(func() {
						unsafeLockFreeBackup = true
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/pre-backup-lock-script-was-run
`)
						instance1.CreateScript("/var/vcap/jobs/redis-broker/bin/bbr/pre-backup-lock", ``)
					})

					It("executes and logs the locks", func() {
						By("running the pre-backup-lock script", func() {
							Expect(instance1.FileExists("/tmp/pre-backup-lock-script-was-run")).To(BeFalse())
						})

						By("logging that it is locking the instance, and listing the scripts", func() {
							assertOutput(session.Out, []string{
								"/var/vcap/jobs/redis-broker/bin/bbr/pre-backup-lock",
								"/var/vcap/jobs/redis/bin/bbr/pre-backup-lock",
								`Skipping lock for deployment`,
							})
						})
					})
				})

				Context("and the post-backup-unlock is present, but the unsafe-lock-free-backup is set", func() {
					BeforeEach(func() {
						unsafeLockFreeBackup = true
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
touch /tmp/post-backup-unlock-script-was-run
`)
						instance1.CreateScript("/var/vcap/jobs/redis-broker/bin/bbr/post-backup-unlock", ``)
					})

					It("executes and logs the locks", func() {
						By("running the pre-backup-lock script", func() {
							Expect(instance1.FileExists("/tmp/post-backup-unlock-script-was-run")).To(BeFalse())
						})

						By("logging that it is locking the instance, and listing the scripts", func() {
							assertOutput(session.Out, []string{
								"/var/vcap/jobs/redis-broker/bin/bbr/post-backup-unlock",
								"/var/vcap/jobs/redis/bin/bbr/post-backup-unlock",
								`Skipping unlock after successful backup for deployment`,
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
						By("running the pre-backup-lock scripts", func() {
							Expect(instance1.FileExists("/tmp/pre-backup-lock-output")).To(BeTrue())
						})

						By("not running the backup script", func() {
							Expect(instance1.FileExists("/tmp/backup-script-was-run")).NotTo(BeTrue())
						})

						By("exiting with the correct error code", func() {
							Expect(session.ExitCode()).To(Equal(4))
						})

						By("logging the error", func() {
							Expect(session.Err).To(gbytes.Say(
								"Error attempting to run pre-backup-lock for job redis on redis-dedicated-node/fake-uuid"))
						})

						By("logging stderr", func() {
							Expect(session.Err).To(gbytes.Say("ultra-baz"))
						})

						By("also running the post-backup-unlock scripts", func() {
							Expect(instance1.FileExists("/tmp/post-backup-unlock-output")).To(BeTrue())
						})

						By("not printing a recommendation to run bbr backup-cleanup", func() {
							Expect(string(session.Err.Contents())).NotTo(ContainSubstring(
								"It is recommended that you run `bbr backup-cleanup`"))
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
							Eventually(session).Should(gbytes.Say("Copying backup -- 3.0M uncompressed -- for job redis on redis-dedicated-node/fake-uuid..."))
						})
					})
				})

				Context("when deployment has a post-backup-unlock script", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
echo "$BBR_AFTER_BACKUP_SCRIPTS_SUCCESSFUL" > /tmp/post-backup-unlock-script-was-run
echo "Unlocking release"`)
					})

					It("prints unlock progress to the screen", func() {
						By("runs the pre-backup-lock scripts", func() {
							Expect(instance1.FileExists("/tmp/post-backup-unlock-script-was-run")).To(BeTrue())
							Expect(strings.TrimSpace(instance1.Run("cat", "/tmp/post-backup-unlock-script-was-run"))).To(Equal("true"))
						})

						By("logging the script action", func() {
							assertOutput(session.Out, []string{
								"Unlocking redis on redis-dedicated-node/fake-uuid",
								"Finished unlocking redis on redis-dedicated-node/fake-uuid.",
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

						By("prints an error", func() {
							Expect(session.Err).To(gbytes.Say("Error attempting to run post-backup-unlock for job redis on redis-dedicated-node/fake-uuid"))
						})

						By("prints stderr", func() {
							Expect(session.Err).To(gbytes.Say("ultra-baz"))
						})

						By("printing a recommendation to run bbr backup-cleanup", func() {
							Expect(session.Err).To(gbytes.Say("It is recommended that you run `bbr backup-cleanup`"))
						})
					})
				})

				Context("but /var/vcap/store is not world-accessible", func() {
					BeforeEach(func() {
						instance1.Run("sudo", "chmod", "700", "/var/vcap/store")
					})

					It("successfully backs up the deployment", func() {
						Expect(session.ExitCode()).To(BeZero())
					})
				})
			})

			Context("and we ask for the manifest to be downloaded", func() {
				BeforeEach(func() {
					downloadManifest = true

					director.VerifyAndMock(AppendBuilders(
						InfoWithBasicAuth(),
						VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
						DownloadManifest(deploymentName, manifest),
						SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
						DownloadManifest(deploymentName, "this is a totally valid yaml"),
						CleanupSSH(deploymentName, "redis-dedicated-node"),
					)...)
				})

				It("downloads the manifest", func() {
					Expect(path.Join(backupDirectory(), "manifest.yml")).To(BeARegularFile())
					Expect(os.ReadFile(path.Join(backupDirectory(), "manifest.yml"))).To(Equal([]byte("this is a totally valid yaml")))
				})
			})
		})

		Context("when there are multiple plausible backup scripts", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				By("creating a dummy backup script")
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh

set -u

printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $BBR_ARTIFACT_DIRECTORY/backupdump2
`)

				By("creating a dummy backup script")
				instance1.CreateScript("/var/vcap/jobs/redis-broker/bin/bbr/backup", `#!/usr/bin/env sh

set -u

printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $BBR_ARTIFACT_DIRECTORY/backupdump2
`)

				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName, manifest),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"))
			})

			It("successfully backs up the deployment", func() {
				By("exiting zero", func() {
					Expect(session.ExitCode()).To(BeZero())
				})

				var redisNodeArchivePath, brokerArchivePath string
				By("creating a backup directory which contains the backup artifacts and a metadata file", func() {
					Expect(backupDirectory()).To(BeADirectory())
					redisNodeArchivePath = artifactFile("redis-dedicated-node-0-redis.tar")
					brokerArchivePath = artifactFile("redis-dedicated-node-0-redis-broker.tar")
					Expect(redisNodeArchivePath).To(BeARegularFile())
					Expect(brokerArchivePath).To(BeARegularFile())
					Expect(metadataFile()).To(BeARegularFile())
				})

				By("including the backup files from the instance", func() {
					redisNodeArchive := OpenTarArchive(redisNodeArchivePath)
					Expect(redisNodeArchive.Files()).To(ConsistOf("backupdump1", "backupdump2"))
					Expect(redisNodeArchive.FileContents("backupdump1")).To(Equal("backupcontent1"))
					Expect(redisNodeArchive.FileContents("backupdump2")).To(Equal("backupcontent2"))

					brokerArchive := OpenTarArchive(brokerArchivePath)
					Expect(brokerArchive.Files()).To(ConsistOf("backupdump1", "backupdump2"))
					Expect(brokerArchive.FileContents("backupdump1")).To(Equal("backupcontent1"))
					Expect(brokerArchive.FileContents("backupdump2")).To(Equal("backupcontent2"))
				})

				By("correctly populating the metadata file", func() {
					metadataContents := ParseMetadata(metadataFile())

					currentTimezone, _ := time.Now().Zone()
					Expect(metadataContents.BackupActivityMetadata.StartTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))
					Expect(metadataContents.BackupActivityMetadata.FinishTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))

					Expect(metadataContents.InstancesMetadata).To(HaveLen(1))
					Expect(metadataContents.InstancesMetadata[0].InstanceName).To(Equal("redis-dedicated-node"))
					Expect(metadataContents.InstancesMetadata[0].InstanceIndex).To(Equal("0"))

					redisArtifact := metadataContents.InstancesMetadata[0].FindArtifact("redis")
					Expect(redisArtifact.Name).To(Equal("redis"))
					Expect(redisArtifact.Checksums).To(HaveLen(2))
					Expect(redisArtifact.Checksums["./backupdump1"]).To(Equal(ShaFor("backupcontent1")))
					Expect(redisArtifact.Checksums["./backupdump2"]).To(Equal(ShaFor("backupcontent2")))

					brokerArtifact := metadataContents.InstancesMetadata[0].FindArtifact("redis-broker")
					Expect(brokerArtifact.Name).To(Equal("redis-broker"))
					Expect(brokerArtifact.Checksums).To(HaveLen(2))
					Expect(brokerArtifact.Checksums["./backupdump1"]).To(Equal(ShaFor("backupcontent1")))
					Expect(brokerArtifact.Checksums["./backupdump2"]).To(Equal(ShaFor("backupcontent2")))

					Expect(metadataContents.CustomArtifactsMetadata).To(BeEmpty())
				})

				By("cleaning up backup artifacts from the remote", func() {
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
				})
			})
		})

		Context("when a deployment can't be backed up", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName, manifest),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

				instance1.CreateExecutableFiles(
					"/var/vcap/jobs/redis/bin/ctl",
				)
			})

			It("exits and displays a message", func() {
				Expect(session.ExitCode()).NotTo(BeZero(), "returns a non-zero exit code")
				Expect(session.Err).To(gbytes.Say("Deployment '"+deploymentName+"' has no backup scripts"),
					"prints an error")
				Expect(possibleBackupDirectories(deploymentName, backupWorkspace)).To(HaveLen(0), "does not create a backup on disk")

				By("not printing a recommendation to run bbr backup-cleanup", func() {
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("It is recommended that you run `bbr backup-cleanup`"))
				})
			})
		})

		Context("when the instance backup script fails", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName, manifest),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/bbr/backup", "echo 'ultra-bar'; (>&2 echo 'ultra-baz'); exit 1",
				)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
echo "$BBR_AFTER_BACKUP_SCRIPTS_SUCCESSFUL" > /tmp/post-backup-unlock-script-was-run
echo "Unlocking release"`)
			})

			It("errors and exits gracefully", func() {
				By("returning exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("running the the post-backup-unlock scripts", func() {
					Expect(instance1.FileExists("/tmp/post-backup-unlock-script-was-run")).To(BeTrue())
					Expect(strings.TrimSpace(instance1.Run("cat", "/tmp/post-backup-unlock-script-was-run"))).To(Equal("false"))
				})

				By("not printing a recommendation to run bbr backup-cleanup", func() {
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("It is recommended that you run `bbr backup-cleanup`"))
				})
			})
		})

		Context("when both the instance backup script and cleanup fail", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName, manifest),
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
					assertOutput(session.Err, []string{
						"Error attempting to run backup for job redis on redis-dedicated-node/fake-uuid",
						"ultra-baz",
						"ultra-foo",
					})
				})

				By("printing a recommendation to run bbr backup-cleanup", func() {
					Expect(session.Err).To(gbytes.Say("It is recommended that you run `bbr backup-cleanup`"))
				})
			})
		})

		Context("when backup succeeds but cleanup fails", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName, manifest),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSHFails(deploymentName, "redis-dedicated-node", "Can't do it mate"),
				)

				instance1.CreateExecutableFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)
			})

			It("exits correctly and prints the error", func() {
				By("returning the correct error code", func() {
					Expect(session.ExitCode()).To(Equal(16))
				})

				By("printing an error", func() {
					Expect(session.Err).To(gbytes.Say("Deployment '" + deploymentName + "' failed while cleaning up with error: "))
				})

				By("including the failure message in error output", func() {
					Expect(session.Err).To(gbytes.Say("Can't do it mate"))
				})

				By("creating a backup on disk", func() {
					Expect(backupDirectory()).To(BeADirectory())
				})

				By("printing a recommendation to run bbr backup-cleanup", func() {
					Expect(session.Err).To(gbytes.Say("It is recommended that you run `bbr backup-cleanup`"))
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
					DownloadManifest(deploymentName, manifest),
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

				By("not printing a recommendation to run bbr backup-cleanup", func() {
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("It is recommended that you run `bbr backup-cleanup`"))
				})
			})
		})

		Context("when the job is disabled", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh
exit 0`)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata",
					`#!/usr/bin/env sh
echo "---
skip_bbr_scripts: true
"`)

				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName, manifest),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)
			})

			It("Should exit say no bbr jobs found", func() {
				By("exiting with an error", func() {
					Expect(session).To(gexec.Exit(1))
				})

				By("printing a helpful error message", func() {
					Expect(session.Out).To(gbytes.Say("DEBUG - Found disabled jobs on instance redis-dedicated-node/fake-uuid jobs: redis"))
					Expect(session.Err).To(gbytes.Say("has no backup scripts"))
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
					Index:   newIndex(0),
					ID:      "fake-uuid",
				},
				{
					IPs:     []string{"10.0.0.2"},
					JobName: secondInstanceGroupName,
					Index:   newIndex(0),
					ID:      "fake-uuid-2",
				},
			}
		}

		Context("one backupable", func() {
			var firstReturnedInstance, secondReturnedInstance *testcluster.Instance

			BeforeEach(func() {
				deploymentName = "my-bigger-deployment"
				firstReturnedInstance = testcluster.NewInstance()
				secondReturnedInstance = testcluster.NewInstance()
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, twoInstancesResponse("redis-dedicated-node", "redis-broker")),
					DownloadManifest(deploymentName, manifest),
					append(SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, firstReturnedInstance),
						SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, secondReturnedInstance)...),
					append(CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-broker")...),
				)
				firstReturnedInstance.CreateExecutableFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)
			})

			AfterEach(func() {
				firstReturnedInstance.DieInBackground()
				secondReturnedInstance.DieInBackground()
			})

			It("backs up deployment successfully", func() {
				Expect(session.ExitCode()).To(BeZero())
				Expect(backupDirectory()).To(BeADirectory())
				Expect(path.Join(backupDirectory(), "/redis-dedicated-node-0-redis.tar")).To(BeARegularFile())
				Expect(path.Join(backupDirectory(), "/redis-broker-0-redis.tar")).ToNot(BeAnExistingFile())
			})

			Context("with ordering on pre-backup-lock specified", func() {
				BeforeEach(func() {
					firstReturnedInstance.CreateScript(
						"/var/vcap/jobs/redis/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/redis-pre-backup-lock-called
exit 0`)
					secondReturnedInstance.CreateScript(
						"/var/vcap/jobs/redis-writer/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/redis-writer-pre-backup-lock-called
exit 0`)
					secondReturnedInstance.CreateScript("/var/vcap/jobs/redis-writer/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
backup_should_be_locked_before:
- job_name: redis
  release: redis
"`)
				})

				It("locks in the specified order", func() {
					redisLockTime := firstReturnedInstance.GetCreatedTime("/tmp/redis-pre-backup-lock-called")
					redisWriterLockTime := secondReturnedInstance.GetCreatedTime("/tmp/redis-writer-pre-backup-lock-called")

					Expect(session.Out).To(gbytes.Say("Detected order: redis-writer should be locked before redis/redis during backup"))

					Expect(string(session.Out.Contents())).NotTo(ContainSubstring("discontinued metadata keys backup_name/restore_name"))

					Expect(redisWriterLockTime < redisLockTime).To(BeTrue(), fmt.Sprintf(
						"Writer locked at %s, which is after the server locked (%s)",
						strings.TrimSuffix(redisWriterLockTime, "\n"),
						strings.TrimSuffix(redisLockTime, "\n")))

				})
			})

			Context("with ordering on pre-backup-lock (where the default ordering would unlock in the wrong order)",
				func() {
					BeforeEach(func() {
						secondReturnedInstance.CreateScript(
							"/var/vcap/jobs/redis/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/redis-pre-backup-lock-called
exit 0`)
						firstReturnedInstance.CreateScript(
							"/var/vcap/jobs/redis-writer/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/redis-writer-pre-backup-lock-called
exit 0`)
						secondReturnedInstance.CreateScript(
							"/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
touch /tmp/redis-post-backup-unlock-called
exit 0`)
						firstReturnedInstance.CreateScript(
							"/var/vcap/jobs/redis-writer/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
touch /tmp/redis-writer-post-backup-unlock-called
exit 0`)
						firstReturnedInstance.CreateScript("/var/vcap/jobs/redis-writer/bin/bbr/metadata",
							`#!/usr/bin/env sh
echo "---
backup_should_be_locked_before:
- job_name: redis
  release: redis
"`)
					})

					It("unlocks in the right order", func() {
						By("unlocking the redis job before unlocking the redis-writer job")
						redisUnlockTime := secondReturnedInstance.GetCreatedTime("/tmp/redis-post-backup-unlock-called")
						redisWriterUnlockTime := firstReturnedInstance.GetCreatedTime("/tmp/redis-writer-post-backup-unlock-called")

						Expect(redisUnlockTime < redisWriterUnlockTime).To(BeTrue(), fmt.Sprintf(
							"Writer unlocked at %s, which is before the server unlocked (%s)",
							strings.TrimSuffix(redisWriterUnlockTime, "\n"),
							strings.TrimSuffix(redisUnlockTime, "\n")))
					})
				})

			Context("but the pre-backup-lock ordering is cyclic", func() {
				BeforeEach(func() {
					firstReturnedInstance.CreateScript(
						"/var/vcap/jobs/redis/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/redis-pre-backup-lock-called
exit 0`)
					firstReturnedInstance.CreateScript(
						"/var/vcap/jobs/redis-writer/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/redis-writer-pre-backup-lock-called
exit 0`)
					firstReturnedInstance.CreateScript("/var/vcap/jobs/redis-writer/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
backup_should_be_locked_before:
- job_name: redis
  release: redis
"`)
					firstReturnedInstance.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata",
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

					By("not creating a local backup artifact", func() {
						Expect(possibleBackupDirectories(deploymentName, backupWorkspace)).To(BeEmpty(),
							"Should quit before creating any local backup artifact.")
					})
				})
			})

			Context("but one is disabled", func() {
				BeforeEach(func() {
					secondReturnedInstance.CreateScript(
						"/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh
exit 0`)

					secondReturnedInstance.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
skip_bbr_scripts: true
"`)
				})

				It("only backups up the enabled instance", func() {
					Expect(session.ExitCode()).To(BeZero())

					Expect(string(session.Buffer().Contents())).To(ContainSubstring("DEBUG - Found disabled jobs on instance redis-broker/fake-uuid-2 jobs: redis"))
					Expect(string(session.Buffer().Contents())).To(ContainSubstring("Backing up redis on redis-dedicated-node/fake-uuid"))
					Expect(string(session.Buffer().Contents())).NotTo(ContainSubstring("Backing up redis on redis-broker/fake-uuid-2"))

				})
			})

			Context("and the backup fails during the drain step", func() {
				BeforeEach(func() {
					firstReturnedInstance.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh
rm -rf /usr/bin/shasum
`)
				})

				It("reports that it failed to create the backup", func() {
					By("failing", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					By("logging the error", func() {
						Expect(session.Out).NotTo(gbytes.Say("Backup created of"))
						Expect(session.Out).To(gbytes.Say("Failed to create backup of %s", deploymentName))
						Expect(string(session.Err.Contents())).To(ContainSubstring("It is recommended that you run `bbr backup-cleanup`"))
					})
				})
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
					DownloadManifest(deploymentName, manifest),
					append(SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, backupableInstance1),
						SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, backupableInstance2)...),
					append(CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-broker")...),
				)

				backupableInstance1.CreateExecutableFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)

				backupableInstance2.CreateExecutableFiles(
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
					Expect(backupDirectory()).To(BeADirectory())
					Expect(path.Join(backupDirectory(), "/redis-dedicated-node-0-redis.tar")).To(BeARegularFile())
					Expect(path.Join(backupDirectory(), "/redis-broker-0-redis.tar")).To(BeARegularFile())
				})

				By("printing the backup progress to the screen", func() {
					assertOutput(session.Out, []string{
						fmt.Sprintf("Starting backup of %s...", deploymentName),
						"Backing up redis on redis-dedicated-node/fake-uuid...",
						"Finished backing up redis on redis-dedicated-node/fake-uuid.",
						"Backing up redis on redis-broker/fake-uuid-2...",
						"Finished backing up redis on redis-broker/fake-uuid-2.",
						"Copying backup --",
						"for job redis on redis-dedicated-node/fake-uuid...",
						"for job redis on redis-broker/fake-uuid-2...",
						"Finished copying backup --",
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
						Expect(string(session.Err.Contents())).To(ContainSubstring("It is recommended that you run `bbr backup-cleanup`"))
					})
				})
			})
		})

		Context("and there is a job property 'backup_one_restore_all' set to true", func() {
			var (
				redisBBRGeneratedArtifactFile                 string
				redisDefaultNonBootstrapArtifactFile          string
				firstReturnedInstance, secondReturnedInstance *testcluster.Instance
			)

			BeforeEach(func() {
				manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 2
  jobs:
  - name: redis-dedicated-node
    release: redis
    properties:
      bbr:
        backup_one_restore_all: true
  - name: redis-writer
    release: redis
  - name: redis-broker
    release: redis
`
				deploymentName = "my-bigger-deployment"
				firstReturnedInstance = testcluster.NewInstance()
				secondReturnedInstance = testcluster.NewInstance()

				twoInstancesInSameGroupResponse := func(instanceGroupName string) []mockbosh.VMsOutput {
					return []mockbosh.VMsOutput{
						{
							IPs:       []string{firstReturnedInstance.Address()},
							JobName:   instanceGroupName,
							Index:     newIndex(0),
							ID:        "fake-uuid-0",
							Bootstrap: true,
						},
						{
							IPs:     []string{secondReturnedInstance.Address()},
							JobName: instanceGroupName,
							Index:   newIndex(1),
							ID:      "fake-uuid-1",
						},
					}
				}

				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, twoInstancesInSameGroupResponse("redis-dedicated-node")),
					DownloadManifest(deploymentName, manifest),
					SetupSSHForAllInstances(deploymentName, "redis-dedicated-node", twoInstancesInSameGroupResponse("redis-dedicated-node"), []*testcluster.Instance{
						firstReturnedInstance, secondReturnedInstance,
					}),
					append(
						CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-dedicated-node")...),
				)
				firstReturnedInstance.CreateScript("/var/vcap/jobs/redis-dedicated-node/bin/bbr/backup", `#!/usr/bin/env sh

set -u
touch /tmp/bootstrapped-backup-script-was-run
printf "bootstrap-backupdump-contents" > $BBR_ARTIFACT_DIRECTORY/bootstrap-backupdump
`)
				secondReturnedInstance.CreateScript("/var/vcap/jobs/redis-dedicated-node/bin/bbr/backup", `#!/usr/bin/env sh

set -u
touch /tmp/backup-script-was-run
`)
			})

			JustBeforeEach(func() {
				redisBBRGeneratedArtifactFile = path.Join(backupDirectory(), "/redis-dedicated-node-redis-backup-one-restore-all.tar")
				redisDefaultNonBootstrapArtifactFile = path.Join(backupDirectory(), "/redis-dedicated-node-1-redis-dedicated-node.tar")
			})

			It("creates a named artifact", func() {
				By("running the backup scripts", func() {
					Expect(firstReturnedInstance.FileExists("/tmp/bootstrapped-backup-script-was-run")).To(BeTrue())
					Expect(secondReturnedInstance.FileExists("/tmp/backup-script-was-run")).To(BeTrue())
				})

				By("creating a deterministically-named backup artifact", func() {
					archive := OpenTarArchive(redisBBRGeneratedArtifactFile)

					Expect(archive.Files()).To(ConsistOf("bootstrap-backupdump"))
					Expect(archive.FileContents("bootstrap-backupdump")).To(Equal("bootstrap-backupdump-contents"))
				})

				By("creating an empty artifact with the default name for the non bootstrap node", func() {
					Expect(redisDefaultNonBootstrapArtifactFile).To(BeARegularFile())

					archive := OpenTarArchive(redisDefaultNonBootstrapArtifactFile)

					Expect(archive.Files()).To(BeEmpty())
				})

				By("recording the artifact as a custom artifact in the backup metadata", func() {
					metadataContents := ParseMetadata(metadataFile())

					currentTimezone, _ := time.Now().Zone()
					Expect(metadataContents.BackupActivityMetadata.StartTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))
					Expect(metadataContents.BackupActivityMetadata.FinishTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))

					Expect(metadataContents.CustomArtifactsMetadata).To(HaveLen(1))
					Expect(metadataContents.CustomArtifactsMetadata[0].Name).To(Equal("redis-dedicated-node-redis-backup-one-restore-all"))
					Expect(metadataContents.CustomArtifactsMetadata[0].Checksums).To(HaveLen(1))
					Expect(metadataContents.CustomArtifactsMetadata[0].Checksums["./bootstrap-backupdump"]).To(Equal(ShaFor("bootstrap-backupdump-contents")))
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
				Expect(session.Err).To(gbytes.Say("Director responded with non-successful status code"))
			})

			By("not printing a recommendation to run bbr backup-cleanup", func() {
				Expect(session.Err).NotTo(gbytes.Say("It is recommended that you run `bbr backup-cleanup`"))
			})
		})
	})
})

var _ = Describe("Backup --all-deployments", func() {
	const deploymentName1 = "little-deployment-1"

	var director *mockhttp.Server
	var backupWorkspace string
	var artifactPath string
	var session *gexec.Session
	var instance1 *testcluster.Instance
	var unsafeLockFreeBackup bool
	var params []string
	manifest := `---
instance_groups:
- name: redis
  instances: 1
  jobs:
  - name: redis
    release: redis
`

	backupDirectory := func(deploymentName, backupWorkspace string) string {
		matches := possibleBackupDirectories(deploymentName, backupWorkspace)

		Expect(matches).To(HaveLen(1), "backup directory not found")
		return path.Join(backupWorkspace, matches[0])
	}

	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = os.MkdirTemp(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())

		artifactPath, err = os.MkdirTemp("/tmp", "artifact-path-")
		Expect(err).NotTo(HaveOccurred())

		instance1 = testcluster.NewInstance()

		unsafeLockFreeBackup = false
	})

	AfterEach(func() {
		director.VerifyMocks()
		director.Close()

		instance1.DieInBackground()
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
		Expect(os.RemoveAll(artifactPath)).To(Succeed())
	})

	Describe("Backup exits gracefully", func() {
		JustBeforeEach(func() {
			params = []string{
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--password", "admin",
				"--target", director.URL,
				"--all-deployments",
				"backup",
				"--artifact-path", artifactPath,
			}
			if unsafeLockFreeBackup {
				params = append(params, "--unsafe-lock-free")
			}
			session = binary.Run(backupWorkspace, []string{}, params...)
		})

		Context("when the deployment is backupable", func() {
			const instanceGroupName = "redis"

			BeforeEach(func() {
				deploymentVMs := func(instanceGroupName string) []mockbosh.VMsOutput {
					return []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: instanceGroupName,
							Index:   newIndex(0),
							ID:      "fake-uuid",
						},
					}
				}

				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deploymentName1}),
					InfoWithBasicAuth(),
					VmsForDeployment(deploymentName1, deploymentVMs(instanceGroupName)),
					DownloadManifest(deploymentName1, manifest),
					SetupSSH(deploymentName1, instanceGroupName, "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName1, instanceGroupName),
				)...)

				instance1.CreateExecutableFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)
			})

			It("backs up successfully", func() {
				By("backing the deployment successfully", func() {
					Expect(session.ExitCode()).To(BeZero())

					deployment1Artifact := backupDirectory(deploymentName1, artifactPath)
					Expect(deployment1Artifact).To(BeADirectory())
					Expect(path.Join(deployment1Artifact, "/redis-0-redis.tar")).To(BeARegularFile())

				})

				By("printing the backup progress to the screen", func() {
					logfilePath := filepath.Join(artifactPath, fmt.Sprintf("%s_%s.log", deploymentName1, `(\d){8}T(\d){6}Z\b`))

					Expect(string(session.Out.Contents())).To(ContainSubstring("Starting backup..."))
					AssertOutputWithTimestamp(session.Out, []string{
						fmt.Sprintf("Pending: %s", deploymentName1),
						fmt.Sprintf("Starting backup of %s, log file: %s", deploymentName1, logfilePath),
						fmt.Sprintf("Finished backup of %s", deploymentName1),
						fmt.Sprintf("Successfully backed up: %s", deploymentName1),
					})
				})

				By("outputing the deployment logs to file", func() {
					files, err := filepath.Glob(filepath.Join(artifactPath, fmt.Sprintf("%s_*.log", deploymentName1)))
					Expect(err).NotTo(HaveOccurred())
					Expect(files).To(HaveLen(1))

					logFilePath := files[0]
					Expect(filepath.Base(logFilePath)).To(MatchRegexp(fmt.Sprintf("%s_%s.log", deploymentName1, `(\d){8}T(\d){6}Z\b`)))

					backupLogContent, err := os.ReadFile(logFilePath) //nolint:ineffassign,staticcheck
					output := string(backupLogContent)

					Expect(output).To(ContainSubstring("INFO - Looking for scripts"))
					Expect(output).To(ContainSubstring("INFO - redis/fake-uuid/redis/backup"))
					Expect(output).To(ContainSubstring(fmt.Sprintf("INFO - Running pre-checks for backup of %s...", deploymentName1)))
					Expect(output).To(ContainSubstring(fmt.Sprintf("INFO - Starting backup of %s...", deploymentName1)))
					Expect(output).To(ContainSubstring("INFO - Running pre-backup-lock scripts..."))
					Expect(output).To(ContainSubstring("INFO - Finished running pre-backup-lock scripts."))
					Expect(output).To(ContainSubstring("INFO - Running backup scripts..."))
					Expect(output).To(ContainSubstring("INFO - Backing up redis on redis/fake-uuid..."))
					Expect(output).To(ContainSubstring("INFO - Finished running backup scripts."))
					Expect(output).To(ContainSubstring("INFO - Running post-backup-unlock scripts..."))
					Expect(output).To(ContainSubstring("INFO - Finished running post-backup-unlock scripts."))
					Expect(output).To(MatchRegexp("INFO - Copying backup -- [^-]*-- for job redis on redis/fake-uuid..."))
					Expect(output).To(ContainSubstring("INFO - Finished copying backup -- for job redis on redis/fake-uuid..."))
					Expect(output).To(ContainSubstring("INFO - Starting validity checks -- for job redis on redis/fake-uuid..."))
					Expect(output).To(ContainSubstring("INFO - Finished validity checks -- for job redis on redis/fake-uuid..."))
				})
			})

		})

		Context("When the backuper fails to get the deployments", func() {
			BeforeEach(func() {
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					DeploymentsFails("oups"),
				)...)
			})

			It("returns an error", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(session.Err).To(gbytes.Say("oups"))
			})
		})

		Context("When a backuper fails to authenticate", func() {
			BeforeEach(func() {
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deploymentName1}),
					InfoWithBasicAuthFails("oups"),
				)...)
			})

			It("returns an error", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(session.Err).To(gbytes.Say("oups"))
			})
		})

		Context("When the backuper fails to authenticate", func() {
			BeforeEach(func() {
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuthFails("oups"),
				)...)
			})

			It("returns an error", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(session.Err).To(gbytes.Say("oups"))
			})
		})

		Context("when the deployment fails to backup", func() {
			const instanceGroupName = "redis"

			BeforeEach(func() {
				deploymentVMs := func(instanceGroupName string) []mockbosh.VMsOutput {
					return []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: instanceGroupName,
							Index:   newIndex(0),
							ID:      "fake-uuid",
						},
					}
				}

				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deploymentName1}),
					InfoWithBasicAuth(),
					VmsForDeployment(deploymentName1, deploymentVMs(instanceGroupName)),
					DownloadManifest(deploymentName1, manifest),
					SetupSSH(deploymentName1, instanceGroupName, "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName1, instanceGroupName),
				)...)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/bbr/backup", "echo 'ultra-bar'; (>&2 echo 'ultra-baz'); exit 1",
				)
			})

			It("alerts me about the deployment failure", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
				assertOutput(session.Out, []string{
					fmt.Sprintf("Starting backup..."), //nolint:staticcheck
					fmt.Sprintf("Pending: %s", deploymentName1),
					fmt.Sprintf("Starting backup of %s", deploymentName1),
					fmt.Sprintf("ERROR: failed to backup %s", deploymentName1),
					fmt.Sprintf("Error backing up redis on redis/fake-uuid"), //nolint:staticcheck
					fmt.Sprintf("Successfully backed up: "),                  //nolint:staticcheck
					fmt.Sprintf("FAILED: %s", deploymentName1),
				})

				assertOutput(session.Err, []string{
					"1 out of 1 deployments cannot be backed up",
					fmt.Sprintf("%s", deploymentName1), //nolint:staticcheck
					"Error attempting to run backup for job redis on redis/fake-uuid: ultra-baz - exit code 1",
				})
			})

		})

		Context("when the backup artifact directory already exists on a deployment", func() {
			const instanceGroupName = "redis"

			BeforeEach(func() {
				deploymentVMs := func(instanceGroupName string) []mockbosh.VMsOutput {
					return []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: instanceGroupName,
							Index:   newIndex(0),
							ID:      "fake-uuid",
						},
					}
				}

				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deploymentName1}),
					InfoWithBasicAuth(),
					VmsForDeployment(deploymentName1, deploymentVMs(instanceGroupName)),
					DownloadManifest(deploymentName1, manifest),
					SetupSSH(deploymentName1, instanceGroupName, "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName1, instanceGroupName),
				)...)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/bbr/backup", "echo 'ultra-bar'; (>&2 echo 'ultra-baz'); exit 1",
				)

				instance1.CreateDir("/var/vcap/store/bbr-backup")

			})

			It("recommends the operator to run backup-cleanup", func() {
				By("failing", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				By("not deleting the existing backup artifact directory", func() {
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
				})

				By("loging which instance has the extant artifact directory", func() {
					Expect(session.Err).To(gbytes.Say("Directory /var/vcap/store/bbr-backup already exists on instance redis/fake-uuid"))
					Expect(string(session.Err.Contents())).To(ContainSubstring("It is recommended that you run `bbr deployment --all-deployments backup-cleanup`"))
				})
			})

		})

		Context("when the deployments fails to unlock", func() {
			const instanceGroupName = "redis"

			BeforeEach(func() {
				deploymentVMs := func(instanceGroupName string) []mockbosh.VMsOutput {
					return []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: instanceGroupName,
							Index:   newIndex(0),
							ID:      "fake-uuid",
						},
					}
				}

				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deploymentName1}),
					InfoWithBasicAuth(),
					VmsForDeployment(deploymentName1, deploymentVMs(instanceGroupName)),
					DownloadManifest(deploymentName1, manifest),
					SetupSSH(deploymentName1, instanceGroupName, "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName1, instanceGroupName),
				)...)

				instance1.CreateExecutableFiles(
					"/var/vcap/jobs/redis/bin/bbr/backup",
				)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
>&2 echo 'I failed'
exit 1`)
			})

			It("fails and prints a backup cleanup advice", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(session.Err).To(gbytes.Say("It is recommended that you run `bbr deployment --all-deployments backup-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."))
			})
		})

		Context("when there are no deployments", func() {
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

		Context("when called with unsafe-lock-free", func() {
			BeforeEach(func() {
				unsafeLockFreeBackup = true
			})

			It("fails", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(session.Err).To(gbytes.Say("Cannot use the --unsafe-lock-free flag in conjunction with the --all-deployments flag"))
			})
		})

	})

	Describe("Backup gets interrupted", func() {
		Context("when the backuper gets killed while backing up", func() {
			const instanceGroupName = "redis"

			BeforeEach(func() {
				deploymentVMs := func(instanceGroupName string) []mockbosh.VMsOutput {
					return []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: instanceGroupName,
							Index:   newIndex(0),
							ID:      "fake-uuid",
						},
					}
				}

				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					Deployments([]string{deploymentName1}),
					InfoWithBasicAuth(),
					VmsForDeployment(deploymentName1, deploymentVMs(instanceGroupName)),
					DownloadManifest(deploymentName1, manifest),
					SetupSSH(deploymentName1, instanceGroupName, "fake-uuid", 0, instance1),
				)...)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env bash
				echo "" > /tmp/backup
				sleep 600
				`,
				)
			})

			JustBeforeEach(func() {
				params = []string{
					"deployment",
					"--ca-cert", sslCertPath,
					"--username", "admin",
					"--password", "admin",
					"--target", director.URL,
					"--all-deployments",
					"backup",
					"--artifact-path", artifactPath,
				}
				session, _ = binary.Start(backupWorkspace, []string{}, params...)
			})

			It("updates the logfile with the progress so far", func() {
				Eventually(func() bool {
					return instance1.FileExists("/tmp/backup")
				}, 2*time.Minute, 5*time.Second).Should(BeTrue())

				Eventually(session.Kill(), 1*time.Minute).Should(gexec.Exit())

				files, err := filepath.Glob(filepath.Join(artifactPath, fmt.Sprintf("%s_*.log", deploymentName1)))
				Expect(err).NotTo(HaveOccurred())
				Expect(files).To(HaveLen(1))

				logFilePath := files[0]
				Expect(filepath.Base(logFilePath)).To(MatchRegexp(fmt.Sprintf("%s_%s.log", deploymentName1, `(\d){8}T(\d){6}Z\b`)))

				backupLogContent, err := os.ReadFile(logFilePath) //nolint:ineffassign,staticcheck
				output := string(backupLogContent)

				Expect(output).To(ContainSubstring("INFO - Looking for scripts"))
				Expect(output).To(ContainSubstring("INFO - redis/fake-uuid/redis/backup"))
				Expect(output).To(ContainSubstring(fmt.Sprintf("INFO - Running pre-checks for backup of %s...", deploymentName1)))
				Expect(output).To(ContainSubstring(fmt.Sprintf("INFO - Starting backup of %s...", deploymentName1)))
				Expect(output).To(ContainSubstring("INFO - Running pre-backup-lock scripts..."))
				Expect(output).To(ContainSubstring("INFO - Finished running pre-backup-lock scripts."))
				Expect(output).To(ContainSubstring("INFO - Running backup scripts..."))
				Expect(output).To(ContainSubstring("INFO - Backing up redis on redis/fake-uuid..."))
				Expect(output).NotTo(ContainSubstring("INFO - Finished running backup scripts."))
			})
		})
	})
})

func assertOutput(b *gbytes.Buffer, strings []string) {
	for _, str := range strings {
		Expect(string(b.Contents())).To(ContainSubstring(str))
	}
}

func possibleBackupDirectories(deploymentName, backupWorkspace string) []string {
	dirs, err := os.ReadDir(backupWorkspace)
	Expect(err).NotTo(HaveOccurred())
	backupDirectoryPattern := regexp.MustCompile(`\b` + deploymentName + `_(\d){8}T(\d){6}Z\b$`)

	matches := []string{}
	for _, dir := range dirs {
		dirName := dir.Name()
		if backupDirectoryPattern.MatchString(dirName) {
			matches = append(matches, dirName)
		}
	}
	return matches
}
