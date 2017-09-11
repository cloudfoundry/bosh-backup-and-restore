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
				fakeJobWithDependencies("first", []JobSpecifier{}),
				fakeJobWithDependencies("second", []JobSpecifier{}),
				fakeJobWithDependencies("third", []JobSpecifier{}),
			}
		})

		It("returns the list of input jobs, untouched", func() {
			orderedJobs, err := directorLockOrderer.Order(jobs)

			Expect(orderedJobs).To(Equal(jobs))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when a job has some locking dependencies", func() {
		BeforeEach(func() {
			jobs = []Job{
				fakeJobWithDependencies("first", []JobSpecifier{}),
				fakeJobWithDependencies("second", []JobSpecifier{{Name: "first"}}),
				fakeJobWithDependencies("third", []JobSpecifier{}),
			}
		})

		It("returns an error", func() {
			orderedJobs, err := directorLockOrderer.Order(jobs)

			Expect(orderedJobs).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("director job 'second' specifies locking dependencies, which are not allowed for director jobs")))
		})
	})
})

func fakeJobWithDependencies(name string, shouldBeLockedBefore []JobSpecifier) *fakes.FakeJob {
	job := new(fakes.FakeJob)
	job.NameReturns(name)
	job.ShouldBeLockedBeforeReturns(shouldBeLockedBefore)
	return job
}
