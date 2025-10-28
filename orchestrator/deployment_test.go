package orchestrator_test

import (
	"fmt"

	"github.com/cloudfoundry/bosh-backup-and-restore/executor"
	executorFakes "github.com/cloudfoundry/bosh-backup-and-restore/executor/fakes"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		var (
			lockError    error
			lockOrderer  *fakes.FakeLockOrderer
			fakeExecutor *executorFakes.FakeExecutor
		)

		BeforeEach(func() {
			lockOrderer = new(fakes.FakeLockOrderer)
			fakeExecutor = new(executorFakes.FakeExecutor)
			instances = []orchestrator.Instance{instance1, instance2, instance3}
			lockOrderer.OrderReturns([][]orchestrator.Job{{job2a}, {job3a, job1a}, {job1b}}, nil)
		})

		JustBeforeEach(func() {
			lockError = deployment.PreBackupLock(lockOrderer, fakeExecutor)
		})

		It("delegates the execution to the executor", func() {
			Expect(lockError).NotTo(HaveOccurred())
			Expect(lockOrderer.OrderArgsForCall(0)).To(ConsistOf(job1a, job1b, job2a, job3a))
			Expect(fakeExecutor.RunArgsForCall(0)).To(Equal([][]executor.Executable{
				{orchestrator.NewJobPreBackupLockExecutable(job2a)},
				{orchestrator.NewJobPreBackupLockExecutable(job3a), orchestrator.NewJobPreBackupLockExecutable(job1a)},
				{orchestrator.NewJobPreBackupLockExecutable(job1b)},
			}))
		})

		Context("if the pre-backup-lock fails", func() {
			BeforeEach(func() {
				fakeExecutor.RunReturns([]error{
					fmt.Errorf("job1b failed"),
					fmt.Errorf("job2a failed"),
				})
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
		var err error
		var fakeExecutor *executorFakes.FakeExecutor

		JustBeforeEach(func() {
			err = deployment.Backup(fakeExecutor)
		})

		BeforeEach(func() {
			instance1.IsBackupableReturns(true)
			instance2.IsBackupableReturns(false)
			instance3.IsBackupableReturns(true)
			instances = []orchestrator.Instance{instance1, instance2, instance3}

			instance1.JobsReturns([]orchestrator.Job{job1a})
			instance2.JobsReturns([]orchestrator.Job{job2a})
			instance3.JobsReturns([]orchestrator.Job{job3a})
			fakeExecutor = new(executorFakes.FakeExecutor)
		})

		It("calls Backup() on all backupable instances", func() {
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeExecutor.RunArgsForCall(0)).To(Equal([][]executor.Executable{
				{orchestrator.NewBackupExecutable(job1a), orchestrator.NewBackupExecutable(job3a)},
			}))
		})

		Context("when backing up an instance fails", func() {
			BeforeEach(func() {
				fakeExecutor.RunReturns([]error{
					fmt.Errorf("backup instance1 failed"),
				})
			})

			It("fails and stops the backup", func() {
				Expect(err).To(MatchError(ContainSubstring("backup instance1 failed")))

				Expect(fakeExecutor.RunArgsForCall(0)).To(Equal([][]executor.Executable{
					{orchestrator.NewBackupExecutable(job1a), orchestrator.NewBackupExecutable(job3a)},
				}))
			})
		})
	})

	Context("PostBackupUnlock", func() {
		var (
			lockError    error
			lockOrderer  *fakes.FakeLockOrderer
			fakeExecutor *executorFakes.FakeExecutor
		)

		BeforeEach(func() {
			lockOrderer = new(fakes.FakeLockOrderer)
			fakeExecutor = new(executorFakes.FakeExecutor)
			instances = []orchestrator.Instance{instance1, instance2, instance3}
			lockOrderer.OrderReturns([][]orchestrator.Job{{job2a}, {job3a, job1a}, {job1b}}, nil)
		})

		It("calls the executor with post successful backup executables", func() {
			lockError = deployment.PostBackupUnlock(true, lockOrderer, fakeExecutor)

			Expect(lockError).NotTo(HaveOccurred())
			Expect(lockOrderer.OrderArgsForCall(0)).To(ConsistOf(job1a, job1b, job2a, job3a))
			Expect(fakeExecutor.RunArgsForCall(0)).To(Equal([][]executor.Executable{
				{orchestrator.NewJobPostSuccessfulBackupUnlockExecutable(job2a)},
				{orchestrator.NewJobPostSuccessfulBackupUnlockExecutable(job3a), orchestrator.NewJobPostSuccessfulBackupUnlockExecutable(job1a)},
				{orchestrator.NewJobPostSuccessfulBackupUnlockExecutable(job1b)},
			}))
		})

		Context("when called after a failed backup", func() {
			It("calls the executor with post failed backup executables", func() {
				lockError = deployment.PostBackupUnlock(false, lockOrderer, fakeExecutor)

				Expect(lockError).NotTo(HaveOccurred())
				Expect(lockOrderer.OrderArgsForCall(0)).To(ConsistOf(job1a, job1b, job2a, job3a))
				Expect(fakeExecutor.RunArgsForCall(0)).To(Equal([][]executor.Executable{
					{orchestrator.NewJobPostFailedBackupUnlockExecutable(job2a)},
					{orchestrator.NewJobPostFailedBackupUnlockExecutable(job3a), orchestrator.NewJobPostFailedBackupUnlockExecutable(job1a)},
					{orchestrator.NewJobPostFailedBackupUnlockExecutable(job1b)},
				}))
			})
		})

		Context("if the post-backup-unlock fails", func() {
			It("fails", func() {
				fakeExecutor.RunReturns([]error{
					fmt.Errorf("job1b failed"),
					fmt.Errorf("job2a failed"),
				})

				lockError = deployment.PostBackupUnlock(true, lockOrderer, fakeExecutor)

				Expect(lockError).To(MatchError(SatisfyAll(
					ContainSubstring("job1b failed"),
					ContainSubstring("job2a failed"),
				)))
			})
		})

		Context("if the lockOrderer returns an error", func() {
			It("fails", func() {
				lockOrderer.OrderReturns(nil, fmt.Errorf("test lock orderer error"))

				lockError = deployment.PostBackupUnlock(true, lockOrderer, fakeExecutor)

				Expect(lockError).To(MatchError(ContainSubstring("test lock orderer error")))
			})
		})
	})

	Context("IsBackupable", func() {
		Context("when at least one instance is backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance2.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(deployment.IsBackupable()).To(BeTrue())
			})
		})

		Context("when no instances are backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance2.IsBackupableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns false", func() {
				Expect(deployment.IsBackupable()).To(BeFalse())
			})
		})
	})

	Context("BackupableInstances", func() {
		BeforeEach(func() {
			instance1.IsBackupableReturns(true)
			instance2.IsBackupableReturns(false)
			instance3.IsBackupableReturns(true)
			instances = []orchestrator.Instance{instance1, instance2, instance3}
		})

		It("returns a list of all backupable instances", func() {
			Expect(deployment.BackupableInstances()).To(ConsistOf(instance1, instance3))
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

	Context("Restore", func() {
		var err error

		JustBeforeEach(func() {
			err = deployment.Restore()
		})

		BeforeEach(func() {
			instance1.IsRestorableReturns(true)
			instance2.IsRestorableReturns(false)
			instance3.IsRestorableReturns(true)
			instances = []orchestrator.Instance{instance1, instance2, instance3}
		})

		It("calls Restore() an all restorable instances", func() {
			Expect(err).NotTo(HaveOccurred())

			Expect(instance1.RestoreCallCount()).To(Equal(1))
			Expect(instance2.RestoreCallCount()).To(Equal(0))
			Expect(instance3.RestoreCallCount()).To(Equal(1))
		})

		Context("when restoring an instance fails", func() {
			BeforeEach(func() {
				instance1.RestoreReturns(fmt.Errorf("and some salt and vinegar crisps"))
			})

			It("fails and stops the restore", func() {
				Expect(err).To(MatchError(fmt.Errorf("and some salt and vinegar crisps")))

				Expect(instance1.RestoreCallCount()).To(Equal(1))
				Expect(instance2.RestoreCallCount()).To(Equal(0))
				Expect(instance3.RestoreCallCount()).To(Equal(0))
			})
		})
	})

	Context("PreRestoreLock", func() {
		var (
			lockError    error
			lockOrderer  *fakes.FakeLockOrderer
			fakeExecutor *executorFakes.FakeExecutor
		)

		BeforeEach(func() {
			lockOrderer = new(fakes.FakeLockOrderer)
			fakeExecutor = new(executorFakes.FakeExecutor)
			instances = []orchestrator.Instance{instance1, instance2, instance3}
			lockOrderer.OrderReturns([][]orchestrator.Job{{job2a}, {job3a, job1a}, {job1b}}, nil)
		})

		JustBeforeEach(func() {
			lockError = deployment.PreRestoreLock(lockOrderer, fakeExecutor)
		})

		It("delegates the execution to the executor", func() {
			Expect(lockError).NotTo(HaveOccurred())
			Expect(lockOrderer.OrderArgsForCall(0)).To(ConsistOf(job1a, job1b, job2a, job3a))
			Expect(fakeExecutor.RunArgsForCall(0)).To(Equal([][]executor.Executable{
				{orchestrator.NewJobPreRestoreLockExecutable(job2a)},
				{orchestrator.NewJobPreRestoreLockExecutable(job3a), orchestrator.NewJobPreRestoreLockExecutable(job1a)},
				{orchestrator.NewJobPreRestoreLockExecutable(job1b)},
			}))
		})

		Context("if the pre-restore-lock fails", func() {
			BeforeEach(func() {
				fakeExecutor.RunReturns([]error{
					fmt.Errorf("job1b failed"),
					fmt.Errorf("job2a failed"),
				})
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

	Context("PostRestoreUnlock", func() {
		var (
			lockError    error
			lockOrderer  *fakes.FakeLockOrderer
			fakeExecutor *executorFakes.FakeExecutor
		)

		BeforeEach(func() {
			lockOrderer = new(fakes.FakeLockOrderer)
			fakeExecutor = new(executorFakes.FakeExecutor)
			instances = []orchestrator.Instance{instance1, instance2, instance3}
			lockOrderer.OrderReturns([][]orchestrator.Job{{job2a}, {job3a, job1a}, {job1b}}, nil)
		})

		JustBeforeEach(func() {
			lockError = deployment.PostRestoreUnlock(lockOrderer, fakeExecutor)
		})

		It("delegates the execution to the executor", func() {
			Expect(lockError).NotTo(HaveOccurred())
			Expect(lockOrderer.OrderArgsForCall(0)).To(ConsistOf(job1a, job1b, job2a, job3a))
			Expect(fakeExecutor.RunArgsForCall(0)).To(Equal([][]executor.Executable{
				{orchestrator.NewJobPostRestoreUnlockExecutable(job2a)},
				{orchestrator.NewJobPostRestoreUnlockExecutable(job3a), orchestrator.NewJobPostRestoreUnlockExecutable(job1a)},
				{orchestrator.NewJobPostRestoreUnlockExecutable(job1b)},
			}))
		})

		Context("if the post-restore-unlock fails", func() {
			BeforeEach(func() {
				fakeExecutor.RunReturns([]error{
					fmt.Errorf("job1b failed"),
					fmt.Errorf("job2a failed"),
				})
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

	Context("IsRestorable", func() {
		Context("when at least one instance is restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instance2.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(deployment.IsRestorable()).To(BeTrue())
			})
		})

		Context("when no instances are restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instance2.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("succeeds and returns false", func() {
				Expect(deployment.IsRestorable()).To(BeFalse())
			})
		})
	})

	Context("RestorableInstances", func() {
		BeforeEach(func() {
			instance1.IsRestorableReturns(true)
			instance2.IsRestorableReturns(false)
			instance3.IsRestorableReturns(true)
			instances = []orchestrator.Instance{instance1, instance2, instance3}
		})

		It("returns a list of all backupable instances", func() {
			Expect(deployment.RestorableInstances()).To(ConsistOf(instance1, instance3))
		})
	})

	Context("Cleanup", func() {
		var err error

		JustBeforeEach(func() {
			err = deployment.Cleanup()
		})

		BeforeEach(func() {
			instances = []orchestrator.Instance{instance1, instance2, instance3}
		})

		It("succeeds and runs cleanup", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(instance1.CleanupCallCount()).To(Equal(1))
			Expect(instance2.CleanupCallCount()).To(Equal(1))
			Expect(instance3.CleanupCallCount()).To(Equal(1))
		})

		Context("When some instances fail to cleanup", func() {

			BeforeEach(func() {
				instance1.CleanupReturns(fmt.Errorf("foo"))
				instance3.CleanupReturns(fmt.Errorf("bar"))
			})

			It("fails, returning all error messages, and continues cleanup of instances", func() {
				Expect(err).To(MatchError(SatisfyAll(
					ContainSubstring("foo"),
					ContainSubstring("bar"),
				)))

				Expect(instance1.CleanupCallCount()).To(Equal(1))
				Expect(instance2.CleanupCallCount()).To(Equal(1))
				Expect(instance3.CleanupCallCount()).To(Equal(1))
			})
		})
	})

	Context("CleanupPrevious", func() {
		var err error

		JustBeforeEach(func() {
			err = deployment.CleanupPrevious()
		})

		BeforeEach(func() {
			instance1.IsBackupableReturns(true)
			instance1.IsRestorableReturns(false)

			instance2.IsBackupableReturns(false)
			instance2.IsRestorableReturns(true)

			instance3.IsBackupableReturns(false)
			instance3.IsRestorableReturns(false)

			instances = []orchestrator.Instance{instance1, instance2, instance3}
		})

		It("calls CleanupPrevious() on all backupable or restorable instances", func() {
			Expect(err).NotTo(HaveOccurred())

			Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
			Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
			Expect(instance3.CleanupPreviousCallCount()).To(Equal(0))
		})

		Context("when cleaning up some instances fails", func() {
			BeforeEach(func() {
				instance1.CleanupPreviousReturns(fmt.Errorf("foo"))
				instance2.CleanupPreviousReturns(fmt.Errorf("bar"))
			})

			It("fails, returning all error messages, and continues cleanup of instances", func() {
				Expect(err).To(MatchError(SatisfyAll(
					ContainSubstring("foo"),
					ContainSubstring("bar"),
				)))

				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance3.CleanupPreviousCallCount()).To(Equal(0))
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
