package jobexecutor

import (
	"errors"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("JobExecutor", func() {
	var firstJob, secondJob, thirdJob *fakes.FakeJob

	BeforeEach(func() {
		firstJob = new(fakes.FakeJob)
		secondJob = new(fakes.FakeJob)
		thirdJob = new(fakes.FakeJob)

		firstJob.NameReturns("first")
		secondJob.NameReturns("second")
		thirdJob.NameReturns("third")

		firstJob.PreBackupLockReturns(errors.New("error from first job"))
		thirdJob.PreBackupLockReturns(errors.New("error from third job"))
	})

	TestJobExecutor := func(jobExecutor orchestrator.JobExecutionStrategy) {
		It("performs a specified behaviour on a list of lists of jobs", func() {
			errs := jobExecutor.Run(orchestrator.JobPreBackupLocker, [][]orchestrator.Job{{firstJob}, {secondJob, thirdJob}})

			By("calling the provided func on each provided job", func() {
				Expect(firstJob.PreBackupLockCallCount()).To(Equal(1))
				Expect(secondJob.PreBackupLockCallCount()).To(Equal(1))
				Expect(thirdJob.PreBackupLockCallCount()).To(Equal(1))
			})

			By("collecting all errors", func() {
				Expect(errs).To(ConsistOf(
					MatchError("error from first job"),
					MatchError("error from third job"),
				))
			})
		})
	}

	TestJobExecutor(NewSerialJobExecutor())
	TestJobExecutor(NewParallelJobExecutor())
})
