package director

import (
	"os"
	"path"
	"strings"

	. "github.com/cloudfoundry/bosh-backup-and-restore/integration"
	"github.com/cloudfoundry/bosh-backup-and-restore/testcluster"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"time"

	"fmt"

	"regexp"

	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup", func() {
	var backupWorkspace string
	var session *gexec.Session
	var stdin io.WriteCloser
	var directorAddress, directorIP string
	var waitForBackupToFinish bool

	possibleBackupDirectories := func() []string {
		dirs, err := os.ReadDir(backupWorkspace)
		Expect(err).NotTo(HaveOccurred())
		backupDirectoryPattern := regexp.MustCompile(`\b` + directorIP + `_(\d){8}T(\d){6}Z\b`)

		matches := []string{}
		for _, dir := range dirs {
			dirName := dir.Name()
			if backupDirectoryPattern.MatchString(dirName) {
				matches = append(matches, dirName)
			}
		}
		return matches
	}

	backupDirectory := func() string {
		matches := possibleBackupDirectories()

		Expect(matches).To(HaveLen(1), "backup directory not found")
		return path.Join(backupWorkspace, matches[0])
	}

	BeforeEach(func() {
		var err error
		backupWorkspace, err = os.MkdirTemp(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
		waitForBackupToFinish = true
	})

	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
	})

	JustBeforeEach(func() {
		env := []string{"BOSH_CLIENT_SECRET=admin"}

		params := []string{
			"director",
			"--host", directorAddress,
			"--username", "foobar",
			"--private-key-path", pathToPrivateKeyFile,
			"--debug",
			"backup",
		}

		if waitForBackupToFinish {
			session = binary.Run(
				backupWorkspace,
				env,
				params...,
			)
		} else {
			session, stdin = binary.Start(
				backupWorkspace,
				env,
				params...,
			)
			Eventually(session).Should(gbytes.Say(".+"))
		}
	})

	Context("When there is a director instance", func() {
		var directorInstance *testcluster.Instance

		BeforeEach(func() {
			directorInstance = testcluster.NewInstance()
			directorInstance.CreateUser("foobar", readFile(pathToPublicKeyFile))
			directorAddress = directorInstance.Address()
			directorIP = directorInstance.IP()
		})

		AfterEach(func() {
			directorInstance.DieInBackground()
		})

		Context("and there is a backup script", func() {
			BeforeEach(func() {
				directorInstance.CreateExecutableFiles("/var/vcap/jobs/bosh/bin/bbr/backup")
			})

			Context("and the backup script succeeds", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/backup", `#!/usr/bin/env sh
set -u
printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $BBR_ARTIFACT_DIRECTORY/backupdump2
`)
				})

				It("successfully backs up the director", func() {
					By("exiting zero", func() {
						Expect(session.ExitCode()).To(BeZero())
					})

					boshBackupFilePath := path.Join(backupDirectory(), "/bosh-0-bosh.tar")
					metadataFilePath := path.Join(backupDirectory(), "/metadata")

					By("creating a backup directory which contains a backup artifact and a metadata file", func() {
						Expect(backupDirectory()).To(BeADirectory())
						Expect(boshBackupFilePath).To(BeARegularFile())
						Expect(metadataFilePath).To(BeARegularFile())
					})

					By("having successfully run the backup script, using the $BBR_ARTIFACT_DIRECTORY variable", func() {
						archive := OpenTarArchive(boshBackupFilePath)

						Expect(archive.Files()).To(ConsistOf("backupdump1", "backupdump2"))
						Expect(archive.FileContents("backupdump1")).To(Equal("backupcontent1"))
						Expect(archive.FileContents("backupdump2")).To(Equal("backupcontent2"))
					})

					By("correctly populating the metadata file", func() {
						metadataContents := ParseMetadata(metadataFilePath)

						currentTimezone, _ := time.Now().Zone()
						Expect(metadataContents.BackupActivityMetadata.StartTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))
						Expect(metadataContents.BackupActivityMetadata.FinishTime).To(MatchRegexp(`^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2}) ` + currentTimezone + "$"))

						Expect(metadataContents.InstancesMetadata).To(HaveLen(1))
						Expect(metadataContents.InstancesMetadata[0].InstanceName).To(Equal("bosh"))
						Expect(metadataContents.InstancesMetadata[0].InstanceIndex).To(Equal("0"))

						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Name).To(Equal("bosh"))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums).To(HaveLen(2))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./backupdump1"]).To(Equal(ShaFor("backupcontent1")))
						Expect(metadataContents.InstancesMetadata[0].Artifacts[0].Checksums["./backupdump2"]).To(Equal(ShaFor("backupcontent2")))

						Expect(metadataContents.CustomArtifactsMetadata).To(BeEmpty())
					})

					By("printing the backup progress to the screen", func() {
						Expect(session.Out).To(gbytes.Say("INFO - Looking for scripts"))
						Expect(session.Out).To(gbytes.Say("INFO - bosh/0/bosh/backup"))
						Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Running pre-checks for backup of %s...", directorIP)))
						Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Starting backup of %s...", directorIP)))
						Expect(session.Out).To(gbytes.Say("INFO - Running pre-backup-lock scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished running pre-backup-lock scripts."))
						Expect(session.Out).To(gbytes.Say("INFO - Running backup scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Backing up bosh on bosh/0..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished running backup scripts."))
						Expect(session.Out).To(gbytes.Say("INFO - Running post-backup-unlock scripts..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished running post-backup-unlock scripts."))
						Expect(session.Out).To(gbytes.Say("INFO - Copying backup -- [^-]*-- for job bosh on bosh/0..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished copying backup -- for job bosh on bosh/0..."))
						Expect(session.Out).To(gbytes.Say("INFO - Starting validity checks -- for job bosh on bosh/0..."))
						Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./backupdump[12]`))
						Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./backupdump[12]`))
						Expect(session.Out).To(gbytes.Say("DEBUG - Calculating shasum for remote files"))
						Expect(session.Out).To(gbytes.Say("DEBUG - Comparing shasums"))
						Expect(session.Out).To(gbytes.Say("INFO - Finished validity checks -- for job bosh on bosh/0..."))
					})

					By("cleaning up backup artifacts from the remote", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
					})
				})
			})

			Context("but the backup script fails", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/backup", "echo 'NOPE!'; exit 1")
				})

				It("fails to backup the director", func() {
					By("returning exit code 1", func() {
						Expect(session.ExitCode()).To(Equal(1))
					})
				})
			})

			Context("but the backup artifact directory already exists", func() {
				BeforeEach(func() {
					directorInstance.CreateDir("/var/vcap/store/bbr-backup")
				})

				It("fails to backup the director", func() {
					By("exiting non-zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					By("printing a log message saying the director instance cannot be backed up", func() {
						Expect(session.Err).To(gbytes.Say("Directory /var/vcap/store/bbr-backup already exists on instance bosh/0"))
					})

					By("not deleting the existing artifact directory", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
					})
				})
			})

			Context("with ordering on pre-backup-lock specified", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
backup_should_be_locked_before:
- job_name: postgres
  release: bosh
"`)
					directorInstance.CreateScript(
						"/var/vcap/jobs/postgres/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/postgres-pre-backup-lock-called
exit 0`)

					directorInstance.CreateScript(
						"/var/vcap/jobs/bosh/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/bosh-pre-backup-lock-called
exit 0`)

					directorInstance.CreateScript(
						"/var/vcap/jobs/postgres/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
touch /tmp/postgres-post-backup-unlock-called
exit 0`)

					directorInstance.CreateScript(
						"/var/vcap/jobs/bosh/bin/bbr/post-backup-unlock", `#!/usr/bin/env sh
touch /tmp/bosh-post-backup-unlock-called
exit 0`)
				})

				It("locks in the specified order", func() {
					Expect(directorInstance.FileExists("/tmp/postgres-pre-backup-lock-called")).To(BeTrue())
					postgresJobLockTime := directorInstance.GetCreatedTime("/tmp/postgres-pre-backup-lock-called")

					Expect(directorInstance.FileExists("/tmp/bosh-pre-backup-lock-called")).To(BeTrue())
					boshJobLockTime := directorInstance.GetCreatedTime("/tmp/bosh-pre-backup-lock-called")

					Expect(session.Out).To(gbytes.Say("Detected order: bosh should be locked before postgres during backup"))

					Expect(boshJobLockTime < postgresJobLockTime).To(BeTrue(), fmt.Sprintf(
						"'bosh' job locked at %s, which is after the 'postgres' job locked (%s)",
						strings.TrimSuffix(boshJobLockTime, "\n"),
						strings.TrimSuffix(postgresJobLockTime, "\n")))
				})

				It("unlocks in the right order", func() {
					By("unlocking the postgres job before unlocking the bosh job")
					postgresJobUnlockTime := directorInstance.GetCreatedTime("/tmp/postgres-post-backup-unlock-called")
					boshJobUnlockTime := directorInstance.GetCreatedTime("/tmp/bosh-post-backup-unlock-called")

					Expect(postgresJobUnlockTime < boshJobUnlockTime).To(BeTrue(), fmt.Sprintf(
						"'bosh' job unlocked at %s, which is before the 'postgres' job unlocked (%s)",
						strings.TrimSuffix(boshJobUnlockTime, "\n"),
						strings.TrimSuffix(postgresJobUnlockTime, "\n")))
				})
			})

			Context("but the pre-backup-lock ordering is cyclic", func() {
				BeforeEach(func() {
					directorInstance.CreateScript(
						"/var/vcap/jobs/bosh/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/bosh-pre-backup-lock-called
exit 0`)
					directorInstance.CreateScript(
						"/var/vcap/jobs/postgres/bin/bbr/pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/postgres-writer-pre-restore-lock-called
exit 0`)
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
backup_should_be_locked_before:
- job_name: postgres
  release: bosh
"`)
					directorInstance.CreateScript("/var/vcap/jobs/postgres/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
backup_should_be_locked_before:
- job_name: bosh
  release: bosh
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
		})

		Context("but there are no backup scripts", func() {
			BeforeEach(func() {
				directorInstance.CreateExecutableFiles("/var/vcap/jobs/bosh/bin/bbr/not-a-backup-script")
			})

			It("fails to backup the director", func() {
				By("returning exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("printing an error", func() {
					Expect(session.Err).To(gbytes.Say(fmt.Sprintf("Deployment '%s' has no backup scripts", directorIP)))
				})

				By("not printing a recommendation to run bbr backup-cleanup", func() {
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("It is recommended that you run `bbr backup-cleanup`"))
				})
			})
		})

		Context("and the bbr process receives SIGINT while backing up", func() {
			BeforeEach(func() {
				waitForBackupToFinish = false

				By("creating a backup script that takes a while")
				directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/backup", `#!/usr/bin/env sh

				set -u

				sleep 5

				printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
				`)
			})

			Context("and the user decides to cancel the backup", func() {
				It("terminates", func() {
					Eventually(session, 10*time.Second).Should(gbytes.Say("Backing up"))
					session.Interrupt()

					By("printing a helpful message and waiting for user input", func() {
						Consistently(session.Exited).ShouldNot(BeClosed(), "bbr exited without user confirmation")
						Eventually(session).Should(gbytes.Say(`Stopping a backup can leave the system in bad state. Are you sure you want to cancel\? \[yes/no\]`))
					})

					stdin.Write([]byte("yes\n")) //nolint:errcheck

					By("then exiting with a failure", func() {
						Eventually(session, 20*time.Second).Should(gexec.Exit(1))
					})

					By("outputting a warning about cleanup", func() {
						Eventually(session).Should(gbytes.Say("It is recommended that you run `bbr backup-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."))
					})

					By("not creating an artifact tar from the interrupted director backup script", func() {
						boshBackupFilePath := path.Join(backupDirectory(), "/bosh-0-bosh.tar")
						Expect(boshBackupFilePath).NotTo(BeAnExistingFile())
					})
				})
			})

			Context("and the user decides to continue backup", func() {
				It("continues to run", func() {
					session.Interrupt()

					By("printing a helpful message and waiting for user input", func() {
						Consistently(session.Exited).ShouldNot(BeClosed(), "bbr process terminated in response to signal")
						Eventually(session).Should(gbytes.Say(`Stopping a backup can leave the system in bad state. Are you sure you want to cancel\? \[yes/no\]`))
						Expect(string(session.Out.Contents())).To(HaveSuffix("[yes/no]\n"))
					})

					stdin.Write([]byte("no\n")) //nolint:errcheck

					By("waiting for the backup to finish successfully", func() {
						Eventually(session, 20).Should(gexec.Exit(0))
					})

					By("still completing the backup", func() {
						boshBackupFilePath := path.Join(backupDirectory(), "/bosh-0-bosh.tar")
						archive := OpenTarArchive(boshBackupFilePath)

						Expect(archive.Files()).To(ConsistOf("backupdump1"))
						Expect(archive.FileContents("backupdump1")).To(Equal("backupcontent1"))
					})
				})
			})
		})
	})

	Context("When the director does not resolve", func() {
		BeforeEach(func() {
			directorAddress = "no:22"
		})

		It("fails to backup the director", func() {
			By("returning exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(session.Err).To(SatisfyAny(
					gbytes.Say("no such host"),
					gbytes.Say("server misbehaving"),
					gbytes.Say("No address associated with hostname"),
					gbytes.Say("Temporary failure in name resolution")))
			})

			By("not printing a recommendation to run bbr backup-cleanup", func() {
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("It is recommended that you run `bbr backup-cleanup`"))
			})
		})
	})
})
