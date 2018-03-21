package orchestrator_test

import (
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Deployment", func() {
	var (
		deployment orchestrator.Deployment
		logger     *fakes.FakeLogger

		instances []orchestrator.Instance
		instance1 *fakes.FakeInstance
		instance2 *fakes.FakeInstance
		instance3 *fakes.FakeInstance

		job1a *fakes.FakeJob
		job1b *fakes.FakeJob
		job2a *fakes.FakeJob
		job3a *fakes.FakeJob
	)

	BeforeEach(func() {
		logger = new(fakes.FakeLogger)

		instance1 = new(fakes.FakeInstance)
		instance2 = new(fakes.FakeInstance)
		instance3 = new(fakes.FakeInstance)

		job1a = new(fakes.FakeJob)
		job1b = new(fakes.FakeJob)
		job2a = new(fakes.FakeJob)
		job3a = new(fakes.FakeJob)

		instance1.JobsReturns([]orchestrator.Job{job1a, job1b})
		instance2.JobsReturns([]orchestrator.Job{job2a})
		instance3.JobsReturns([]orchestrator.Job{job3a})
	})

	JustBeforeEach(func() {
		deployment = orchestrator.NewDeployment(logger, instances)
	})

	Context("PreBackupLock", func() {
		var lockError error
		var lockOrderer *fakes.FakeLockOrderer

		var orderedListOfLockedJobs []string
		var preBackupLockOrderedStub = func(jobName string) func() error {
			return func() error {
				orderedListOfLockedJobs = append(orderedListOfLockedJobs, jobName)
				return nil
			}
		}

		BeforeEach(func() {
			lockOrderer = new(fakes.FakeLockOrderer)

			orderedListOfLockedJobs = []string{}

			job1a.PreBackupLockStub = preBackupLockOrderedStub("job1a")
			job1b.PreBackupLockStub = preBackupLockOrderedStub("job1b")
			job2a.PreBackupLockStub = preBackupLockOrderedStub("job2a")
			job3a.PreBackupLockStub = preBackupLockOrderedStub("job3a")

			instances = []orchestrator.Instance{instance1, instance2, instance3}

			lockOrderer.OrderReturns([][]orchestrator.Job{{job2a, job3a, job1a, job1b}}, nil)
		})

		JustBeforeEach(func() {
			lockError = deployment.PreBackupLock(lockOrderer, executor.NewSerialExecutor())
		})

		It("succeeds", func() {
			Expect(lockError).NotTo(HaveOccurred())

			By("locking the jobs in the order specified by the orderer", func() {
				Expect(lockOrderer.OrderArgsForCall(0)).To(ConsistOf(job1a, job1b, job2a, job3a))

				Expect(job1a.PreBackupLockCallCount()).To(Equal(1))
				Expect(job1b.PreBackupLockCallCount()).To(Equal(1))
				Expect(job2a.PreBackupLockCallCount()).To(Equal(1))
				Expect(job3a.PreBackupLockCallCount()).To(Equal(1))

				Expect(orderedListOfLockedJobs).To(Equal([]string{"job2a", "job3a", "job1a", "job1b"}))
			})
		})

		Context("if the pre-backup-lock fails", func() {
			BeforeEach(func() {
				job1b.PreBackupLockReturns(fmt.Errorf("job1b failed"))
				job2a.PreBackupLockReturns(fmt.Errorf("job2a failed"))
			})

			It("fails", func() {
				Expect(lockError).To(MatchError(SatisfyAll(
					ContainSubstring("job1b failed"),
					ContainSubstring("job2a failed"),
				)))
			})
		})

		Context("if the lockOrderer returns an error", func() {
			BeforeEach(func() {
				lockOrderer.OrderReturns(nil, fmt.Errorf("test lock orderer error"))
			})

			It("fails", func() {
				Expect(lockError).To(MatchError(ContainSubstring("test lock orderer error")))
			})
		})
	})

	Context("Backup", func() {
		var (
			backupError error
		)
		JustBeforeEach(func() {
			backupError = deployment.Backup()
		})

		Context("Single instance, backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})
			It("succeeds and backs up the instance", func() {
				Expect(backupError).NotTo(HaveOccurred())
				Expect(instance1.BackupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, all backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("succeeds and backs up all the instances", func() {
				Expect(backupError).NotTo(HaveOccurred())

				Expect(instance1.BackupCallCount()).To(Equal(1))
				Expect(instance2.BackupCallCount()).To(Equal(1))
			})
		})
		Context("Multiple instances, some backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("succeeds", func() {
				Expect(backupError).NotTo(HaveOccurred())

				By("backing up the only the backupable instance", func() {
					Expect(instance1.BackupCallCount()).To(Equal(1))
				})
				By("not backing up the non backupable instance", func() {
					Expect(instance2.BackupCallCount()).To(Equal(0))
				})
			})
		})

		Context("Multiple instances, some failing to backup", func() {
			BeforeEach(func() {
				backupError := fmt.Errorf("very clever sandwich")
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)
				instance1.BackupReturns(backupError)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("fails and stops the backup", func() {
				Expect(backupError).To(MatchError("very clever sandwich"))

				Expect(instance1.BackupCallCount()).To(Equal(1))
				Expect(instance2.BackupCallCount()).To(Equal(0))
			})
		})
	})

	Context("PostBackupUnlock", func() {
		var unlockError, expectedError error
		var lockOrderer *fakes.FakeLockOrderer
		var orderedListOfUnlockedJobs []string
		var postBackupUnlockOrderedStub = func(jobName string) func() error {
			return func() error {
				orderedListOfUnlockedJobs = append(orderedListOfUnlockedJobs, jobName)
				return nil
			}
		}

		BeforeEach(func() {
			lockOrderer = new(fakes.FakeLockOrderer)

			orderedListOfUnlockedJobs = []string{}

			job1a.PostBackupUnlockStub = postBackupUnlockOrderedStub("job1a")
			job1b.PostBackupUnlockStub = postBackupUnlockOrderedStub("job1b")
			job2a.PostBackupUnlockStub = postBackupUnlockOrderedStub("job2a")
			job3a.PostBackupUnlockStub = postBackupUnlockOrderedStub("job3a")

			instances = []orchestrator.Instance{instance1, instance2, instance3}

			lockOrderer.OrderReturns([][]orchestrator.Job{{job2a}, {job3a, job1a}, {job1b}}, nil)

			expectedError = fmt.Errorf("something went terribly wrong")
		})

		JustBeforeEach(func() {
			unlockError = deployment.PostBackupUnlock(lockOrderer, executor.NewSerialExecutor())
		})

		It("succeeds", func() {
			Expect(unlockError).NotTo(HaveOccurred())

			By("unlocking the jobs in the reverse order to that specified by the orderer", func() {
				Expect(lockOrderer.OrderArgsForCall(0)).To(ConsistOf(job1a, job1b, job2a, job3a))

				Expect(job1a.PostBackupUnlockCallCount()).To(Equal(1))
				Expect(job1b.PostBackupUnlockCallCount()).To(Equal(1))
				Expect(job2a.PostBackupUnlockCallCount()).To(Equal(1))
				Expect(job3a.PostBackupUnlockCallCount()).To(Equal(1))

				Expect(orderedListOfUnlockedJobs).To(Equal([]string{"job1b", "job3a", "job1a", "job2a"}))
			})
		})

		Context("if the post-backup-unlock fails", func() {
			BeforeEach(func() {
				job1b.PostBackupUnlockReturns(fmt.Errorf("job1b failed"))
				job2a.PostBackupUnlockReturns(fmt.Errorf("job2a failed"))
			})

			It("fails", func() {
				Expect(unlockError).To(MatchError(SatisfyAll(
					ContainSubstring("job1b failed"),
					ContainSubstring("job2a failed"),
				)))
			})
		})

		Context("if the lockOrderer returns an error", func() {
			BeforeEach(func() {
				lockOrderer.OrderReturns(nil, fmt.Errorf("test lock orderer error"))
			})

			It("fails", func() {
				Expect(unlockError).To(MatchError(ContainSubstring("test lock orderer error")))
			})
		})
	})

	Context("IsBackupable", func() {
		var IsBackupable bool

		JustBeforeEach(func() {
			IsBackupable = deployment.IsBackupable()
		})

		Context("Single instance with a backup script", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})

			It("checks for backup scripts and returns true", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
				Expect(IsBackupable).To(BeTrue())
			})
		})

		Context("Single instance, no backup script", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})

			It("checks if the instance has a backup script", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
			})

			It("returns true", func() {
				Expect(IsBackupable).To(BeFalse())
			})
		})

		Context("Multiple instances, some with backup scripts", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance2.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
				Expect(instance2.IsBackupableCallCount()).To(Equal(1))
				Expect(IsBackupable).To(BeTrue())
			})
		})

		Context("Multiple instances, none with backup scripts", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance2.IsBackupableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns false", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
				Expect(instance2.IsBackupableCallCount()).To(Equal(1))
				Expect(IsBackupable).To(BeFalse())
			})
		})
	})

	Context("CheckArtifactDir", func() {
		var artifactDirError error

		BeforeEach(func() {
			instance1.NameReturns("foo")
			instance1.IDReturns("0")

			instance2.NameReturns("bar")
			instance2.IDReturns("0")

			instance1.ArtifactDirExistsReturns(false, nil)
			instance2.ArtifactDirExistsReturns(false, nil)
			instances = []orchestrator.Instance{instance1, instance2}
		})

		JustBeforeEach(func() {
			artifactDirError = deployment.CheckArtifactDir()
		})

		Context("when artifact directory does not exist", func() {
			It("does not fail", func() {
				Expect(artifactDirError).NotTo(HaveOccurred())
			})
		})

		Context("when artifact directory exists", func() {
			BeforeEach(func() {
				instance1.ArtifactDirExistsReturns(true, nil)
				instance2.ArtifactDirExistsReturns(true, nil)
			})

			It("fails and the error includes the names of the instances on which the directory exists", func() {
				Expect(artifactDirError).To(MatchError(SatisfyAll(
					ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance foo/0"),
					ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance bar/0"),
				)))
			})
		})

		Context("when call to check artifact directory fails", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1}
				instance1.ArtifactDirExistsReturns(false, fmt.Errorf("oh dear"))
			})

			It("fails and the error includes the names of the instances on which the error occurred", func() {
				Expect(artifactDirError.Error()).To(ContainSubstring("Error checking /var/vcap/store/bbr-backup on instance foo/0"))
			})
		})
	})

	Context("CustomArtifactNamesMatch", func() {
		var artifactMatchError error

		JustBeforeEach(func() {
			artifactMatchError = deployment.CustomArtifactNamesMatch()
		})

		BeforeEach(func() {
			instances = []orchestrator.Instance{instance1, instance2}
		})

		Context("when the custom names match", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1"})
				instance2.CustomRestoreArtifactNamesReturns([]string{"custom1"})
			})

			It("is nil", func() {
				Expect(artifactMatchError).NotTo(HaveOccurred())
			})
		})

		Context("when the multiple custom names match", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1"})
				instance1.CustomRestoreArtifactNamesReturns([]string{"custom2"})
				instance2.CustomBackupArtifactNamesReturns([]string{"custom2"})
				instance2.CustomRestoreArtifactNamesReturns([]string{"custom1"})
			})

			It("is nil", func() {
				Expect(artifactMatchError).NotTo(HaveOccurred())
			})
		})
		Context("when the custom dont match", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1"})
				instance2.NameReturns("job2Name")
				instance2.CustomRestoreArtifactNamesReturns([]string{"custom2"})
			})

			It("to return an error", func() {
				Expect(artifactMatchError).To(MatchError("The job2Name restore script expects a backup script which produces custom2 artifact which is not present in the deployment."))
			})
		})
	})

	Context("HasUniqueCustomArtifactNames", func() {
		var isValid bool

		JustBeforeEach(func() {
			isValid = deployment.HasUniqueCustomArtifactNames()
		})

		Context("Single instance, with unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1", "custom2"})
				instances = []orchestrator.Instance{instance1}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})

		Context("Single instance, with non-unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"the-same", "the-same"})
				instances = []orchestrator.Instance{instance1}
			})

			It("returns false", func() {
				Expect(isValid).To(BeFalse())
			})
		})

		Context("multiple instances, with unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1", "custom2"})
				instance2.CustomBackupArtifactNamesReturns([]string{"custom3", "custom4"})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})

		Context("multiple instances, with non-unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1", "custom2"})
				instance2.CustomBackupArtifactNamesReturns([]string{"custom2", "custom4"})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns false", func() {
				Expect(isValid).To(BeFalse())
			})
		})

		Context("multiple instances, with no metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{})
				instance2.CustomBackupArtifactNamesReturns([]string{})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})
	})

	Context("Restore", func() {
		var (
			restoreError error
		)

		JustBeforeEach(func() {
			restoreError = deployment.Restore()
		})

		Context("Single instance, restoreable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})

			It("succeeds", func() {
				Expect(restoreError).NotTo(HaveOccurred())

				By("restoring the instance", func() {
					Expect(instance1.RestoreCallCount()).To(Equal(1))
				})

				By("logging before restoring", func() {
					_, message, _ := logger.InfoArgsForCall(0)
					Expect(message).To(Equal("Running restore scripts..."))
				})
			})
		})

		Context("Single instance, not restoreable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})

			It("succeeds and does not restore the instance", func() {
				Expect(restoreError).NotTo(HaveOccurred())
				Expect(instance1.RestoreCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, all restoreable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instance2.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("succeeds and restores all instances", func() {
				Expect(restoreError).NotTo(HaveOccurred())
				Expect(instance1.RestoreCallCount()).To(Equal(1))
				Expect(instance2.RestoreCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, some restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instance2.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("succeeds", func() {
				Expect(restoreError).NotTo(HaveOccurred())

				By("restoring only the restorable instance", func() {
					Expect(instance1.RestoreCallCount()).To(Equal(1))
				})

				By("not restoring the non restorable instance", func() {
					Expect(instance2.RestoreCallCount()).To(Equal(0))
				})
			})
		})

		Context("Multiple instances, some failing to restore", func() {
			var restoreError = fmt.Errorf("and some salt and vinegar crisps")

			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instance2.IsRestorableReturns(true)
				instance1.RestoreReturns(restoreError)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				Expect(restoreError).To(MatchError(restoreError))
			})

			It("stops invoking backup, after the error", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(1))
				Expect(instance2.RestoreCallCount()).To(Equal(0))
			})
		})
	})

	Context("PreRestoreLock", func() {
		var lockError error
		var lockOrderer *fakes.FakeLockOrderer

		var orderedListOfLockedJobs []string
		var preRestoreLockOrderedStub = func(jobName string) func() error {
			return func() error {
				orderedListOfLockedJobs = append(orderedListOfLockedJobs, jobName)
				return nil
			}
		}

		BeforeEach(func() {
			lockOrderer = new(fakes.FakeLockOrderer)

			orderedListOfLockedJobs = []string{}

			job1a.PreRestoreLockStub = preRestoreLockOrderedStub("job1a")
			job1b.PreRestoreLockStub = preRestoreLockOrderedStub("job1b")
			job2a.PreRestoreLockStub = preRestoreLockOrderedStub("job2a")
			job3a.PreRestoreLockStub = preRestoreLockOrderedStub("job3a")

			instances = []orchestrator.Instance{instance1, instance2, instance3}

			lockOrderer.OrderReturns([][]orchestrator.Job{{job2a, job3a, job1a, job1b}}, nil)
		})

		JustBeforeEach(func() {
			lockError = deployment.PreRestoreLock(lockOrderer, executor.NewSerialExecutor())
		})

		It("locks the jobs in the order specified by the orderer", func() {
			Expect(lockOrderer.OrderArgsForCall(0)).To(ConsistOf(job1a, job1b, job2a, job3a))

			Expect(job1a.PreRestoreLockCallCount()).To(Equal(1))
			Expect(job1b.PreRestoreLockCallCount()).To(Equal(1))
			Expect(job2a.PreRestoreLockCallCount()).To(Equal(1))
			Expect(job3a.PreRestoreLockCallCount()).To(Equal(1))

			Expect(orderedListOfLockedJobs).To(Equal([]string{"job2a", "job3a", "job1a", "job1b"}))
		})

		Context("when some jobs fail to PreRestoreLock", func() {
			BeforeEach(func() {
				job1a.PreRestoreLockReturns(errors.New("job 1a failed to lock"))
				job2a.PreRestoreLockReturns(errors.New("job 2a failed to lock"))
			})

			It("fails", func() {
				By("returning a helpful error", func() {
					Expect(lockError).To(MatchError(SatisfyAll(
						ContainSubstring("job 1a failed to lock"),
						ContainSubstring("job 2a failed to lock"),
					)))
				})
			})
		})

		Context("if the lockOrderer returns an error", func() {
			BeforeEach(func() {
				lockOrderer.OrderReturns(nil, fmt.Errorf("test lock orderer error"))
			})

			It("fails", func() {
				Expect(lockError).To(MatchError(ContainSubstring("test lock orderer error")))
			})
		})
	})

	Context("PostRestoreUnlock", func() {
		var unlockError error
		var lockOrderer *fakes.FakeLockOrderer

		var orderedListOfLockedJobs []string
		var postRestoreUnlockOrderedStub = func(jobName string) func() error {
			return func() error {
				orderedListOfLockedJobs = append(orderedListOfLockedJobs, jobName)
				return nil
			}
		}

		BeforeEach(func() {
			lockOrderer = new(fakes.FakeLockOrderer)

			orderedListOfLockedJobs = []string{}

			job1a.PostRestoreUnlockStub = postRestoreUnlockOrderedStub("job1a")
			job1b.PostRestoreUnlockStub = postRestoreUnlockOrderedStub("job1b")
			job2a.PostRestoreUnlockStub = postRestoreUnlockOrderedStub("job2a")
			job3a.PostRestoreUnlockStub = postRestoreUnlockOrderedStub("job3a")

			instances = []orchestrator.Instance{instance1, instance2, instance3}

			lockOrderer.OrderReturns([][]orchestrator.Job{{job2a}, {job3a, job1a}, {job1b}}, nil)
		})

		JustBeforeEach(func() {
			unlockError = deployment.PostRestoreUnlock(lockOrderer, executor.NewSerialExecutor())
		})

		It("unlocks the jobs in the reverse order to that specified by the orderer", func() {
			Expect(lockOrderer.OrderArgsForCall(0)).To(ConsistOf(job1a, job1b, job2a, job3a))

			Expect(job1a.PostRestoreUnlockCallCount()).To(Equal(1))
			Expect(job1b.PostRestoreUnlockCallCount()).To(Equal(1))
			Expect(job2a.PostRestoreUnlockCallCount()).To(Equal(1))
			Expect(job3a.PostRestoreUnlockCallCount()).To(Equal(1))

			Expect(orderedListOfLockedJobs).To(Equal([]string{"job1b", "job3a", "job1a", "job2a"}))
		})

		Context("when some jobs fail to PostRestoreUnlock", func() {
			BeforeEach(func() {
				job1a.PostRestoreUnlockReturns(errors.New("job 1a failed to unlock"))
				job2a.PostRestoreUnlockReturns(errors.New("job 2a failed to unlock"))
			})

			It("fails", func() {
				By("returning a helpful error", func() {
					Expect(unlockError).To(MatchError(SatisfyAll(
						ContainSubstring("job 1a failed to unlock"),
						ContainSubstring("job 2a failed to unlock"),
					)))
				})
			})
		})

		Context("if the lockOrderer returns an error", func() {
			BeforeEach(func() {
				lockOrderer.OrderReturns(nil, fmt.Errorf("test lock orderer error"))
			})

			It("fails", func() {
				Expect(unlockError).To(MatchError(ContainSubstring("test lock orderer error")))
			})
		})
	})

	Context("IsRestorable", func() {
		var (
			isRestorableError error
			isRestorable      bool
		)
		JustBeforeEach(func() {
			isRestorable = deployment.IsRestorable()
		})

		Context("Single instance, restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})

			It("succeeds and returns true", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
				Expect(isRestorable).To(BeTrue())
			})
		})
		Context("Single instance, not restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})

			It("succeeds and returns false", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
				Expect(isRestorable).To(BeFalse())
			})
		})

		Context("Multiple instances, some restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instance2.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("succeeds and returns true", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
				Expect(instance2.IsRestorableCallCount()).To(Equal(1))
				Expect(isRestorable).To(BeTrue())
			})
		})

		Context("Multiple instances, none restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instance2.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("succeeds and returns false", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
				Expect(instance2.IsRestorableCallCount()).To(Equal(1))
				Expect(isRestorable).To(BeFalse())
			})
		})
	})

	Context("Cleanup", func() {
		var (
			actualCleanupError error
		)
		JustBeforeEach(func() {
			actualCleanupError = deployment.Cleanup()
		})

		Context("Single instance", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1}
			})
			It("succeeds and runs cleanup", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
				Expect(instance1.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("succeeds and runs cleanup", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
				Expect(instance1.CleanupCallCount()).To(Equal(1))
				Expect(instance2.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, some failing", func() {
			var cleanupError1 = fmt.Errorf("foo")

			BeforeEach(func() {
				instance1.CleanupReturns(cleanupError1)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails and continues cleanup of instances", func() {
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
				Expect(instance1.CleanupCallCount()).To(Equal(1))
				Expect(instance2.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, all failing", func() {
			var cleanupError1 = fmt.Errorf("foo")
			var cleanupError2 = fmt.Errorf("bar")
			BeforeEach(func() {
				instance1.CleanupReturns(cleanupError1)
				instance2.CleanupReturns(cleanupError2)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				By("returning both error messages", func() {
					Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
					Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError2.Error())))
				})

				By("continuing cleanup of instances", func() {
					Expect(instance1.CleanupCallCount()).To(Equal(1))
					Expect(instance2.CleanupCallCount()).To(Equal(1))
				})
			})
		})
	})

	Context("CleanupPrevious", func() {
		var (
			actualCleanupError error
		)
		JustBeforeEach(func() {
			actualCleanupError = deployment.CleanupPrevious()
		})

		Context("Single instance is backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})
			It("succeeds and cleans up the instance", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
			})
		})

		Context("Single instance is restorable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance1.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})
			It("succeeds and cleans up the instance", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
			})
		})

		Context("Single instance is neither backupable nor restorable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})
			It("succeeds and does not run cleanup", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(0))
			})
		})

		Context("Multiple backupable instances", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.IsRestorableReturns(false)

				instance2.IsBackupableReturns(true)
				instance2.IsRestorableReturns(false)

				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("succeeds and cleanup all the instances", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
			})
		})

		Context("Multiple restoable instances", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance1.IsRestorableReturns(true)

				instance2.IsBackupableReturns(true)
				instance2.IsRestorableReturns(true)

				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("succeeds and cleanup all the instances", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances neither backupable nor restoable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance1.IsRestorableReturns(false)

				instance2.IsBackupableReturns(false)
				instance2.IsRestorableReturns(false)

				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("succeeds and does not run cleanup", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(0))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances some backuable and restorable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.IsRestorableReturns(false)

				instance2.IsBackupableReturns(false)
				instance2.IsRestorableReturns(true)

				instance3.IsBackupableReturns(false)
				instance3.IsRestorableReturns(false)

				instances = []orchestrator.Instance{instance1, instance2, instance3}
			})
			It("succeeds and calls cleanup on the instances", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance3.CleanupPreviousCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, some failing", func() {
			var cleanupError1 = fmt.Errorf("foo")

			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.CleanupPreviousReturns(cleanupError1)
				instance2.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails and continues cleanup of instances", func() {
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, all failing", func() {
			var cleanupError1 = fmt.Errorf("foo")
			var cleanupError2 = fmt.Errorf("bar")
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.CleanupPreviousReturns(cleanupError1)
				instance2.IsRestorableReturns(true)
				instance2.CleanupPreviousReturns(cleanupError2)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				By("accumulating both error messages", func() {
					Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
					Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError2.Error())))
				})

				By("continuing cleanup of instances", func() {
					Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
					Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
				})
			})
		})
	})

	Context("Instances", func() {
		BeforeEach(func() {
			instances = []orchestrator.Instance{instance1, instance2, instance3}
		})
		It("returns instances for the deployment", func() {
			Expect(deployment.Instances()).To(ConsistOf(instance1, instance2, instance3))
		})
	})
})
