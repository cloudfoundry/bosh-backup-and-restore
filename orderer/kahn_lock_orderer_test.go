package orderer

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("KahnLockOrderer", func() {
	Context("new tests", func() {
		type lockingTestCase struct {
			inputJobs   []Job
			orderedJobs []Job
		}

		lockOrderer := NewKahnLockOrderer()

		DescribeTable("counting substring matches",
			func(testCaseBuilder func() lockingTestCase) {
				testCase := testCaseBuilder()
				Expect(lockOrderer.Order(testCase.inputJobs)).To(Equal(testCase.orderedJobs))
			},

			Entry("no jobs", func() lockingTestCase {
				return lockingTestCase{
					inputJobs:   []Job{},
					orderedJobs: []Job{},
				}
			}),

			Entry("one job", func() lockingTestCase {
				var job = NewTestJob("test", []JobSpecifier{})

				return lockingTestCase{
					inputJobs:   []Job{job},
					orderedJobs: []Job{job},
				}
			}),

			Entry("one job, dependency on non-existent job", func() lockingTestCase {
				var job = NewTestJob("test", []JobSpecifier{{Name: "non-existent"}})

				return lockingTestCase{
					inputJobs:   []Job{job},
					orderedJobs: []Job{job},
				}
			}),

			Entry("multiple jobs, no dependencies", func() lockingTestCase {
				var a = NewTestJob("a", []JobSpecifier{})
				var b = NewTestJob("b", []JobSpecifier{})
				var c = NewTestJob("c", []JobSpecifier{})

				return lockingTestCase{
					inputJobs:   []Job{a, b, c},
					orderedJobs: []Job{a, b, c},
				}
			}),

			Entry("multiple jobs, single dependency", func() lockingTestCase {
				var a = NewTestJob("a", []JobSpecifier{})
				var b = NewTestJob("b", []JobSpecifier{{Name: "c"}})
				var c = NewTestJob("c", []JobSpecifier{})

				return lockingTestCase{
					inputJobs:   []Job{a, c, b},
					orderedJobs: []Job{a, b, c},
				}
			}),

			Entry("multiple jobs, dependency on non-existent job", func() lockingTestCase {
				var a = NewTestJob("a", []JobSpecifier{})
				var b = NewTestJob("b", []JobSpecifier{{Name: "e"}})
				var c = NewTestJob("c", []JobSpecifier{})

				return lockingTestCase{
					inputJobs:   []Job{a, b, c},
					orderedJobs: []Job{a, b, c},
				}
			}),

			Entry("multiple jobs, double dependency", func() lockingTestCase {
				var a = NewTestJob("a", []JobSpecifier{})
				var b = NewTestJob("b", []JobSpecifier{{Name: "c"}, {Name: "d"}})
				var c = NewTestJob("c", []JobSpecifier{})
				var d = NewTestJob("d", []JobSpecifier{})

				return lockingTestCase{
					inputJobs:   []Job{a, c, d, b},
					orderedJobs: []Job{a, b, c, d},
				}
			}),

			Entry("multiple jobs, chain of dependencies", func() lockingTestCase {
				var a = NewTestJob("a", []JobSpecifier{{Name: "b"}})
				var b = NewTestJob("b", []JobSpecifier{{Name: "c"}})
				var c = NewTestJob("c", []JobSpecifier{})

				return lockingTestCase{
					inputJobs:   []Job{c, b, a},
					orderedJobs: []Job{a, b, c},
				}
			}),

			Entry("multiple jobs, multiple instances of the same dependee", func() lockingTestCase {
				var a = NewTestJob("a", []JobSpecifier{})
				var b = NewTestJob("b", []JobSpecifier{{Name: "c"}})
				var c1 = NewTestJob("c", []JobSpecifier{})
				var c2 = NewTestJob("c", []JobSpecifier{})
				var c3 = NewTestJob("c", []JobSpecifier{})

				return lockingTestCase{
					inputJobs:   []Job{c1, c2, c3, a, b},
					orderedJobs: []Job{a, b, c1, c2, c3},
				}
			}),

			Entry("multiple jobs, multiple instances of the same dependent", func() lockingTestCase {
				var a = NewTestJob("a", []JobSpecifier{})
				var b1 = NewTestJob("b", []JobSpecifier{{Name: "c"}})
				var b2 = NewTestJob("b", []JobSpecifier{{Name: "c"}})
				var b3 = NewTestJob("b", []JobSpecifier{{Name: "c"}})
				var c = NewTestJob("c", []JobSpecifier{})

				return lockingTestCase{
					inputJobs:   []Job{a, c, b1, b2, b3},
					orderedJobs: []Job{a, b1, b2, b3, c},
				}
			}),
		)
	})
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
