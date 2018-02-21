package jobexecutor

import (
	"errors"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("JobExecutionStrategy", func() {
	var errs []error
	var jobExecutor = NewSerialJobExecutor()
	var firstJob = new(fakes.FakeJob)
	var secondJob = new(fakes.FakeJob)
	var thirdJob = new(fakes.FakeJob)

	BeforeEach(func() {
		firstJob.PreBackupLockReturns(errors.New("error from first job"))
		thirdJob.PreBackupLockReturns(errors.New("error from third job"))
	})

	JustBeforeEach(func() {
		errs = jobExecutor.Run(orchestrator.JobPreBackupLocker, [][]orchestrator.Job{{firstJob}, {secondJob, thirdJob}})
	})

	It("performs a specified behaviour on a list of lists of jobs", func() {
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
})
