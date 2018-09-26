package orchestrator_test

import (
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("JobExecutables", func() {
	var (
		fakeJob    *fakes.FakeJob
		err        error
		executable executor.Executable
	)

	BeforeEach(func() {
		fakeJob = new(fakes.FakeJob)
	})

	Context("JobPreBackupLockExecutor", func() {
		BeforeEach(func() {
			executable = orchestrator.NewJobPreBackupLockExecutable(fakeJob)
		})
		JustBeforeEach(func() {
			err = executable.Execute()
		})

		It("executes pre backup lock", func() {
			Expect(fakeJob.PreBackupLockCallCount()).To(Equal(1))
		})

		Context("when the pre backup lock fails", func() {
			BeforeEach(func() {
				fakeJob.PreBackupLockReturns(fmt.Errorf("fake error"))
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(ContainSubstring("fake error")))
			})
		})
	})

	Context("JobPostBackupUnlockExecutor", func() {
		BeforeEach(func() {
			executable = orchestrator.NewJobPostBackupUnlockExecutable(fakeJob)
		})
		JustBeforeEach(func() {
			err = executable.Execute()
		})

		It("executes pre backup lock", func() {
			Expect(fakeJob.PostBackupUnlockCallCount()).To(Equal(1))
		})

		Context("when the pre backup lock fails", func() {
			BeforeEach(func() {
				fakeJob.PostBackupUnlockReturns(fmt.Errorf("fake error"))
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(ContainSubstring("fake error")))
			})
		})

	})

	Context("JobPreRestoreLockExecutor", func() {
		BeforeEach(func() {
			executable = orchestrator.NewJobPreRestoreLockExecutable(fakeJob)
		})
		JustBeforeEach(func() {
			err = executable.Execute()
		})

		It("executes pre backup lock", func() {
			Expect(fakeJob.PreRestoreLockCallCount()).To(Equal(1))
		})

		Context("when the pre backup lock fails", func() {
			BeforeEach(func() {
				fakeJob.PreRestoreLockReturns(fmt.Errorf("fake error"))
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(ContainSubstring("fake error")))
			})
		})

	})

	Context("JobPostRestoreUnlockExecutor", func() {
		BeforeEach(func() {
			executable = orchestrator.NewJobPostRestoreUnlockExecutable(fakeJob)
		})
		JustBeforeEach(func() {
			err = executable.Execute()
		})

		It("executes pre backup lock", func() {
			Expect(fakeJob.PostRestoreUnlockCallCount()).To(Equal(1))
		})

		Context("when the pre backup lock fails", func() {
			BeforeEach(func() {
				fakeJob.PostRestoreUnlockReturns(fmt.Errorf("fake error"))
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(ContainSubstring("fake error")))
			})
		})

	})
})
