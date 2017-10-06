package orderer

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DirectorLockOrderer", func() {
	var directorLockOrderer = NewDirectorLockOrderer()

	var jobs []Job

	Context("when no job has any locking dependency", func() {
		BeforeEach(func() {
			jobs = []Job{
				fakeJobWithDependencies("first", []JobSpecifier{}, []JobSpecifier{}),
				fakeJobWithDependencies("second", []JobSpecifier{}, []JobSpecifier{}),
				fakeJobWithDependencies("third", []JobSpecifier{}, []JobSpecifier{}),
			}
		})

		It("returns the list of input jobs, untouched", func() {
			orderedJobs, err := directorLockOrderer.Order(jobs)

			Expect(orderedJobs).To(Equal(jobs))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when a job has some backup locking backup dependencies", func() {
		BeforeEach(func() {
			jobs = []Job{
				fakeJobWithDependencies("first", []JobSpecifier{}, []JobSpecifier{}),
				fakeJobWithDependencies("second", []JobSpecifier{{Name: "first"}}, []JobSpecifier{}),
				fakeJobWithDependencies("third", []JobSpecifier{}, []JobSpecifier{}),
			}
		})

		It("returns an error", func() {
			orderedJobs, err := directorLockOrderer.Order(jobs)

			Expect(orderedJobs).To(BeNil())
			Expect(err).To(MatchError(ContainSubstring("director job 'second' specifies locking dependencies, which are not allowed for director jobs")))
		})
	})

	Context("when a job has some restore locking dependencies", func() {
		BeforeEach(func() {
			jobs = []Job{
				fakeJobWithDependencies("first", []JobSpecifier{}, []JobSpecifier{}),
				fakeJobWithDependencies("second", []JobSpecifier{}, []JobSpecifier{{Name: "first"}}),
				fakeJobWithDependencies("third", []JobSpecifier{}, []JobSpecifier{}),
			}
		})

		It("returns an error", func() {
			orderedJobs, err := directorLockOrderer.Order(jobs)

			Expect(orderedJobs).To(BeNil())
			Expect(err).To(MatchError(ContainSubstring("director job 'second' specifies locking dependencies, which are not allowed for director jobs")))
		})
	})
})

func fakeJobWithDependencies(name string, backupShouldBeLockedBefore, restoreShouldBeLockedBefore []JobSpecifier) *fakes.FakeJob {
	job := new(fakes.FakeJob)
	job.NameReturns(name)
	job.BackupShouldBeLockedBeforeReturns(backupShouldBeLockedBefore)
	job.RestoreShouldBeLockedBeforeReturns(restoreShouldBeLockedBefore)
	return job
}
