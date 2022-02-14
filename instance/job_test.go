package instance_test

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	"log"

	"fmt"

	sshfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Job", func() {
	var job instance.Job
	var jobScripts instance.BackupAndRestoreScripts
	var metadata instance.Metadata
	var logOutput *gbytes.Buffer
	var logger boshlog.Logger
	var releaseName string
	var remoteRunner *sshfakes.FakeRemoteRunner
	var instanceIdentifier = "instance/identifier"
	var backupOneRestoreAll bool
	var onBootstrapNode bool

	BeforeEach(func() {
		jobScripts = instance.BackupAndRestoreScripts{
			"/var/vcap/jobs/jobname/bin/bbr/restore",
			"/var/vcap/jobs/jobname/bin/bbr/backup",
			"/var/vcap/jobs/jobname/bin/bbr/pre-backup-lock",
			"/var/vcap/jobs/jobname/bin/bbr/post-backup-unlock",
		}
		metadata = instance.Metadata{}
		logOutput = gbytes.NewBuffer()
		stdoutLog := log.New(logOutput, "[instance-test] ", log.Lshortfile)
		logger = boshlog.New(boshlog.LevelDebug, stdoutLog)
		releaseName = "redis"
		remoteRunner = new(sshfakes.FakeRemoteRunner)
		backupOneRestoreAll = false
		onBootstrapNode = false
	})

	JustBeforeEach(func() {
		job = instance.NewJob(remoteRunner, instanceIdentifier, logger, releaseName, jobScripts, metadata, backupOneRestoreAll, onBootstrapNode)
	})

	Describe("BackupArtifactDirectory", func() {
		It("calculates the artifact directory based on the name", func() {
			Expect(job.BackupArtifactDirectory()).To(Equal("/var/vcap/store/bbr-backup/jobname"))
		})

		Context("when an artifact name is provided", func() {
			BeforeEach(func() {
				backupOneRestoreAll = true
				onBootstrapNode = true
			})

			It("calculates the artifact directory based on the artifact name", func() {
				Expect(job.BackupArtifactDirectory()).To(Equal("/var/vcap/store/bbr-backup/jobname-redis-backup-one-restore-all"))
			})
		})
	})

	Describe("RestoreArtifactDirectory", func() {
		It("calculates the artifact directory based on the name", func() {
			Expect(job.BackupArtifactDirectory()).To(Equal("/var/vcap/store/bbr-backup/jobname"))
		})

		Context("when an artifact name is provided", func() {
			var jobWithName instance.Job

			JustBeforeEach(func() {
				jobWithName = instance.NewJob(
					remoteRunner,
					"",
					logger,
					releaseName,
					jobScripts,
					instance.Metadata{},
					true,
					onBootstrapNode)
			})

			It("calculates the artifact directory based on the artifact name", func() {
				Expect(jobWithName.RestoreArtifactDirectory()).To(Equal("/var/vcap/store/bbr-backup/jobname-redis-backup-one-restore-all"))
			})
		})
	})

	Describe("BackupArtifactName", func() {
		Context("the job has the `bbr.backup_one_restore_all` property enabled", func() {
			BeforeEach(func() {
				backupOneRestoreAll = true
			})

			Context("and the job is on the bootstrap node", func() {
				BeforeEach(func() {
					onBootstrapNode = true
				})

				It("returns a backup artifact name of the form <job>-<release>-backup-one-restore-all", func() {
					Expect(job.BackupArtifactName()).To(Equal("jobname-redis-backup-one-restore-all"))
				})
			})

			Context("and the job is not on the bootstrap node", func() {
				BeforeEach(func() {
					onBootstrapNode = false
				})

				It("returns the default backup artifact name", func() {
					Expect(job.BackupArtifactName()).To(Equal(""))
				})
			})
		})

		Context("the job has a custom backup artifact name", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					BackupName: "fool",
				}
			})

			It("returns the job's custom backup artifact name", func() {
				Expect(job.BackupArtifactName()).To(Equal("fool"))
			})

			Context("the job has the `bbr.backup_one_restore_all` property enabled", func() {
				BeforeEach(func() {
					backupOneRestoreAll = true
				})

				Context("and the job is on the bootstrap node", func() {
					BeforeEach(func() {
						onBootstrapNode = true
					})

					It("returns the backup-one-restore-all artifact name instead of the custom metadata name", func() {
						Expect(job.BackupArtifactName()).To(Equal("jobname-redis-backup-one-restore-all"))
					})
				})

				Context("and the job is not on the bootstrap node", func() {
					It("returns the job's custom backup artifact name", func() {
						Expect(job.BackupArtifactName()).To(Equal("fool"))
					})
				})

			})
		})

		Context("the job does not have a custom backup artifact name", func() {
			It("returns empty string", func() {
				Expect(job.BackupArtifactName()).To(Equal(""))
			})
		})
	})

	Describe("HasMetadataRestoreName", func() {
		Context("when it has RestoreName in metadata", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "some-restore-name",
				}
			})

			It("is true", func() {
				Expect(job.HasMetadataRestoreName()).To(BeTrue())
			})
		})

		Context("when it has RestoreName in metadata", func() {
			It("is false", func() {
				Expect(job.HasMetadataRestoreName()).To(BeFalse())
			})
		})
	})

	Describe("RestoreArtifactName", func() {
		Context("the job has the `bbr.backup_one_restore_all` property enabled", func() {
			BeforeEach(func() {
				backupOneRestoreAll = true
			})

			It("returns a restore artifact name of the form <job>-<release>-backup-one-restore-all", func() {
				Expect(job.RestoreArtifactName()).To(Equal("jobname-redis-backup-one-restore-all"))
			})
		})

		Context("the job has a custom backup artifact name", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "bard",
				}
			})

			It("returns the job's custom backup artifact name", func() {
				Expect(job.RestoreArtifactName()).To(Equal("bard"))
			})
		})

		Context("the job does not have a custom backup artifact name", func() {
			It("returns empty string", func() {
				Expect(job.RestoreArtifactName()).To(Equal(""))
			})
		})
	})

	Describe("HasBackup", func() {
		It("returns true", func() {
			Expect(job.HasBackup()).To(BeTrue())
		})

		Context("no backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/jobname/bin/bbr/restore"}
			})

			It("returns false", func() {
				Expect(job.HasBackup()).To(BeFalse())
			})
		})
	})

	Describe("RestoreScript", func() {
		It("returns the restore script", func() {
			Expect(job.RestoreScript()).To(Equal(instance.Script("/var/vcap/jobs/jobname/bin/bbr/restore")))
		})

		Context("no restore scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/jobname/bin/bbr/backup"}
			})

			It("returns nil", func() {
				Expect(job.RestoreScript()).To(BeEmpty())
			})
		})
	})

	Describe("HasRestore", func() {
		It("returns true", func() {
			Expect(job.HasRestore()).To(BeTrue())
		})

		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/jobname/bin/bbr/backup"}
			})

			It("returns false", func() {
				Expect(job.HasRestore()).To(BeFalse())
			})
		})
	})

	Describe("HasNamedBackupArtifact", func() {
		It("returns false", func() {
			Expect(job.HasNamedBackupArtifact()).To(BeFalse())
		})

		Context("when the job has a named restore artifact", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "whatever",
				}
			})

			It("returns false", func() {
				Expect(job.HasNamedBackupArtifact()).To(BeFalse())
			})
		})

		Context("when the job has a backupAllRestoreOne property set to true", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{}
				backupOneRestoreAll = true
			})

			Context("and the job is on the bootstrap node", func() {
				BeforeEach(func() {
					onBootstrapNode = true
				})

				It("returns true", func() {
					Expect(job.HasNamedBackupArtifact()).To(BeTrue())
				})
			})

			Context("and the job is not on the bootstrap node", func() {
				It("returns true", func() {
					Expect(job.HasNamedBackupArtifact()).To(BeFalse())
				})
			})

		})
	})

	Describe("HasNamedRestoreArtifact", func() {
		It("returns false", func() {
			Expect(job.HasNamedRestoreArtifact()).To(BeFalse())
		})

		Context("when the job has a named restore artifact", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "whatever",
				}
			})

			It("returns true", func() {
				Expect(job.HasNamedRestoreArtifact()).To(BeTrue())
			})
		})

		Context("when the job has a named backup artifact", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					BackupName: "whatever",
				}
			})

			It("returns false", func() {
				Expect(job.HasNamedRestoreArtifact()).To(BeFalse())
			})
		})

		Context("when the job has a backupAllRestoreOne property set to true", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{}
				backupOneRestoreAll = true
			})

			It("returns true", func() {
				Expect(job.HasNamedRestoreArtifact()).To(BeTrue())
			})
		})
	})

	Describe("Backup", func() {
		var backupError error

		JustBeforeEach(func() {
			backupError = job.Backup()
		})

		Context("job has no backup script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/pre-backup-lock",
				}
			})

			It("should not run anything remote runner", func() {
				Expect(remoteRunner.Invocations()).To(HaveLen(0))
			})
		})

		Context("job has a backup script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/backup",
				}
			})

			It("uses the remote runnerto run the script", func() {
				Expect(remoteRunner.CreateDirectoryCallCount()).To(Equal(1))
				Expect(remoteRunner.RunScriptWithEnvCallCount()).To(Equal(1))

				Expect(remoteRunner.CreateDirectoryArgsForCall(0)).To(Equal("/var/vcap/store/bbr-backup/jobname"))
				specifiedScriptPath, specifiedEnvVars, _, _ := remoteRunner.RunScriptWithEnvArgsForCall(0)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/jobname/bin/bbr/backup"))
				Expect(specifiedEnvVars).To(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("ARTIFACT_DIRECTORY", "/var/vcap/store/bbr-backup/jobname/"),
					HaveKeyWithValue("BBR_ARTIFACT_DIRECTORY", "/var/vcap/store/bbr-backup/jobname/"),
				))
			})

			Context("backup script runs successfully", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptWithEnvReturns("stdout", nil)
				})

				It("succeeds", func() {
					Expect(backupError).NotTo(HaveOccurred())
				})
			})

			Context("backup script fails", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptWithEnvReturns("", fmt.Errorf("some weird error"))
				})

				It("fails", func() {
					Expect(backupError).To(MatchError(ContainSubstring("some weird error")))
				})
			})
		})
	})

	Describe("Restore", func() {
		var restoreError error

		JustBeforeEach(func() {
			restoreError = job.Restore()
		})

		Context("job has no restore script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/pre-backup-lock",
				}
			})

			It("should not run anything on the remote runner", func() {
				Expect(remoteRunner.Invocations()).To(HaveLen(0))
			})
		})

		Context("job has a restore script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/restore",
				}
			})

			It("uses the remote runner to run the script", func() {
				Expect(remoteRunner.RunScriptWithEnvCallCount()).To(Equal(1))

				specifiedScriptPath, specifiedEnvVars, _, _ := remoteRunner.RunScriptWithEnvArgsForCall(0)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/jobname/bin/bbr/restore"))
				Expect(specifiedEnvVars).To(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("ARTIFACT_DIRECTORY", "/var/vcap/store/bbr-backup/jobname/"),
					HaveKeyWithValue("BBR_ARTIFACT_DIRECTORY", "/var/vcap/store/bbr-backup/jobname/"),
				))
			})

			Context("restore script runs successfully", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptWithEnvReturns("", nil)
				})

				It("succeeds", func() {
					Expect(restoreError).NotTo(HaveOccurred())
				})
			})

			Context("restore script fails", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptWithEnvReturns("", fmt.Errorf("it went wrong"))
				})

				It("fails", func() {
					Expect(restoreError).To(MatchError(ContainSubstring("it went wrong")))
				})
			})

		})
	})

	Describe("PreBackupLock", func() {
		var preBackupLockError error

		JustBeforeEach(func() {
			preBackupLockError = job.PreBackupLock()
		})

		Context("job has no pre-backup-lock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/restore",
				}
			})

			It("should not call the remote runner", func() {
				Expect(remoteRunner.Invocations()).To(HaveLen(0))
			})
		})

		Context("job has a pre-backup-lock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/pre-backup-lock",
				}
			})

			It("runs the script", func() {
				By("calling the remote runner", func() {
					Expect(remoteRunner.RunScriptCallCount()).To(Equal(1))
					cmd, _ := remoteRunner.RunScriptArgsForCall(0)
					Expect(cmd).To(Equal("/var/vcap/jobs/jobname/bin/bbr/pre-backup-lock"))
				})

				By("logging the script path", func() {
					Expect(string(logOutput.Contents())).To(ContainSubstring(`> /var/vcap/jobs/jobname/bin/bbr/pre-backup-lock`))
				})

				By("logging the job name that it has locked", func() {
					Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
						"INFO - Locking jobname on %s",
						instanceIdentifier,
					)))
					Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
						"INFO - Finished locking jobname on %s",
						instanceIdentifier,
					)))
				})
			})

			Context("pre-backup-lock script runs successfully", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptReturns("stdout", nil)
				})

				It("succeeds", func() {
					Expect(preBackupLockError).NotTo(HaveOccurred())
				})
			})

			Context("pre-backup-lock script errors", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptReturns("", fmt.Errorf("some strange error"))
				})

				It("fails", func() {
					By("including the error in the returned error", func() {
						Expect(preBackupLockError).To(MatchError(ContainSubstring("some strange error")))
					})
				})
			})
		})
	})

	Describe("PostBackupUnlock", func() {
		var (
			postBackupUnlockError error
			afterSuccessfulBackup bool
		)

		BeforeEach(func() {
			afterSuccessfulBackup = true
		})

		JustBeforeEach(func() {
			postBackupUnlockError = job.PostBackupUnlock(afterSuccessfulBackup)
		})

		Context("job has no post-backup-unlock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/restore",
				}
			})

			It("should not run anything on the remote runner", func() {
				Expect(remoteRunner.Invocations()).To(HaveLen(0))
			})
		})

		Context("job has a post-backup-unlock script", func() {
			Context("and is called after a successful backup", func() {
				BeforeEach(func() {
					jobScripts = instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/jobname/bin/bbr/post-backup-unlock",
					}
				})

				It("uses remote runner to run the script", func() {
					Expect(remoteRunner.RunScriptWithEnvCallCount()).To(Equal(1))
					cmd, envVars, _, _ := remoteRunner.RunScriptWithEnvArgsForCall(0)
					Expect(cmd).To(Equal("/var/vcap/jobs/jobname/bin/bbr/post-backup-unlock"))
					Expect(envVars).To(HaveKeyWithValue("BBR_AFTER_BACKUP_SCRIPTS_SUCCESSFUL", "true"))
				})

				Context("post-backup-unlock script runs successfully", func() {
					BeforeEach(func() {
						remoteRunner.RunScriptWithEnvReturns("stdout", nil)
					})

					It("succeeds", func() {
						Expect(postBackupUnlockError).NotTo(HaveOccurred())
					})
				})
			})

			Context("and is called after a failed backup", func() {
				BeforeEach(func() {
					jobScripts = instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/jobname/bin/bbr/post-backup-unlock",
					}
					afterSuccessfulBackup = false
				})

				It("uses remote runner to run the script", func() {
					Expect(remoteRunner.RunScriptWithEnvCallCount()).To(Equal(1))
					cmd, envVars, _, _ := remoteRunner.RunScriptWithEnvArgsForCall(0)
					Expect(cmd).To(Equal("/var/vcap/jobs/jobname/bin/bbr/post-backup-unlock"))
					Expect(envVars).To(HaveKeyWithValue("BBR_AFTER_BACKUP_SCRIPTS_SUCCESSFUL", "false"))
				})

			})

			Context("post-backup-unlock script fails", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptWithEnvReturns("", fmt.Errorf("it failed"))
				})

				It("fails", func() {
					Expect(postBackupUnlockError).To(MatchError(ContainSubstring("it failed")))
				})
			})
		})
	})

	Describe("Release", func() {
		It("returns the job's release name", func() {
			Expect(job.Release()).To(Equal("redis"))
		})
	})

	Describe("PreRestoreLock", func() {
		var preRestoreLockError error

		JustBeforeEach(func() {
			preRestoreLockError = job.PreRestoreLock()
		})

		Context("job has no pre-restore-lock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/restore",
				}
			})

			It("should not run anything on the remote runner", func() {
				Expect(remoteRunner.Invocations()).To(HaveLen(0))
			})
		})

		Context("job has a pre-restore-lock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/pre-restore-lock",
				}
			})

			It("runs the script", func() {
				By("using the remote runner", func() {
					Expect(remoteRunner.RunScriptCallCount()).To(Equal(1))
					cmd, _ := remoteRunner.RunScriptArgsForCall(0)
					Expect(cmd).To(Equal("/var/vcap/jobs/jobname/bin/bbr/pre-restore-lock"))
				})

				By("logging the script path", func() {
					Expect(string(logOutput.Contents())).To(ContainSubstring(`> /var/vcap/jobs/jobname/bin/bbr/pre-restore-lock`))
				})

				By("logging the job name that it has locked", func() {
					Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
						"INFO - Locking jobname on %s",
						instanceIdentifier,
					)))
					Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
						"INFO - Finished locking jobname on %s",
						instanceIdentifier,
					)))
				})
			})

			Context("pre-restore-lock script runs successfully", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptReturns("stdout", nil)
				})

				It("succeeds", func() {
					Expect(preRestoreLockError).NotTo(HaveOccurred())
				})
			})

			Context("pre-restore-lock script fails", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptReturns("", fmt.Errorf("some strange error"))
				})

				It("fails", func() {
					By("including the error in the returned error", func() {
						Expect(preRestoreLockError).To(MatchError(ContainSubstring("some strange error")))
					})
				})
			})
		})
	})

	Describe("PostRestoreUnlock", func() {
		var postRestoreUnlockError error

		JustBeforeEach(func() {
			postRestoreUnlockError = job.PostRestoreUnlock()
		})

		Context("job has no post-restore-unlock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/restore",
				}
			})

			It("should not run anything on the remote runner", func() {
				Expect(remoteRunner.Invocations()).To(HaveLen(0))
			})
		})

		Context("job has a post-restore-unlock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/jobname/bin/bbr/post-restore-unlock",
				}
			})

			It("uses the remote runner to run the script", func() {
				Expect(remoteRunner.RunScriptCallCount()).To(Equal(1))
				cmd, _ := remoteRunner.RunScriptArgsForCall(0)
				Expect(cmd).To(Equal("/var/vcap/jobs/jobname/bin/bbr/post-restore-unlock"))
			})

			Context("post-restore-unlock script runs successfully", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptReturns("stdout", nil)
				})

				It("succeeds", func() {
					Expect(postRestoreUnlockError).NotTo(HaveOccurred())
				})
			})

			Context("post-restore-unlock script fails", func() {
				BeforeEach(func() {
					remoteRunner.RunScriptReturns("", fmt.Errorf("oh no"))
				})

				It("fails", func() {
					Expect(postRestoreUnlockError).To(MatchError(ContainSubstring("oh no")))
				})
			})
		})
	})
})
