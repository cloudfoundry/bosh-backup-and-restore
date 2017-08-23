package orderer

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("KahnLockOrderer", func() {
	var lockOrderer KahnLockOrderer
	var orderedJobs []Job
	var unorderedJobs []Job

	BeforeEach(func() {
		lockOrderer = NewKahnLockOrderer()
	})

	JustBeforeEach(func() {
		orderedJobs = lockOrderer.Order(unorderedJobs)
	})

	Context("when there are no jobs", func() {
		BeforeEach(func() {
			unorderedJobs = []Job{}
		})

		It("returns an empty list", func() {
			Expect(orderedJobs).To(BeEmpty())
		})
	})

	XContext("when there is one job", func() {
		var job = NewTestJob("test", []JobSpecifier{})

		BeforeEach(func() {
			unorderedJobs = []Job{job}
		})

		It("returns a list with that job", func() {
			Expect(orderedJobs).To(ConsistOf(job))
		})
	})

	XContext("when there are two jobs in the wrong order", func() {
		var first = NewTestJob("first", []JobSpecifier{{Name: "second"}})
		var second = NewTestJob("second", []JobSpecifier{})

		BeforeEach(func() {
			unorderedJobs = []Job{second, first}
		})

		It("returns a list with that job", func() {
			Expect(orderedJobs).To(Equal([]Job{first, second}))
		})
	})

	Context("when there is more than one instance with a job that specifies other jobs to lock after it (e.g. many BAMs)", func() {})
	Context("when the job that is specified in a tobelockedbefore actually appears on multiple instances (e.g. many CCAPIs", func() {})
})

// Add id field to Job for testing, otherwise jobs with the same name appear to be equal, and tests pass for the wrong
// reason.
type TestJob struct {
	id                   int64
	name                 string
	shouldBeLockedBefore []JobSpecifier
}

func NewTestJob(name string, shouldBeLockedBefore []JobSpecifier) TestJob {
	return TestJob{name: name, shouldBeLockedBefore: shouldBeLockedBefore}
}

func (job TestJob) Name() string {
	return job.name
}

func (job TestJob) ShouldBeLockedBefore() []JobSpecifier {
	return job.shouldBeLockedBefore
}

func (TestJob) HasBackup() bool                  { panic("implement me") }
func (TestJob) HasRestore() bool                 { panic("implement me") }
func (TestJob) HasNamedBackupArtifact() bool     { panic("implement me") }
func (TestJob) HasNamedRestoreArtifact() bool    { panic("implement me") }
func (TestJob) BackupArtifactName() string       { panic("implement me") }
func (TestJob) RestoreArtifactName() string      { panic("implement me") }
func (TestJob) Backup() error                    { panic("implement me") }
func (TestJob) PreBackupLock() error             { panic("implement me") }
func (TestJob) PostBackupUnlock() error          { panic("implement me") }
func (TestJob) Restore() error                   { panic("implement me") }
func (TestJob) PostRestoreUnlock() error         { panic("implement me") }
func (TestJob) BackupArtifactDirectory() string  { panic("implement me") }
func (TestJob) RestoreArtifactDirectory() string { panic("implement me") }
