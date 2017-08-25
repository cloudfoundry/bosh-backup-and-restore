package orderer

import (
	"strconv"
	"time"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("KahnLockOrderer", func() {
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
			var job = fakeJob("test", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{job},
				orderedJobs: []Job{job},
			}
		}),

		Entry("one job, dependency on non-existent job", func() lockingTestCase {
			var job = fakeJob("test", []JobSpecifier{{Name: "non-existent"}})

			return lockingTestCase{
				inputJobs:   []Job{job},
				orderedJobs: []Job{job},
			}
		}),

		Entry("multiple jobs, no dependencies", func() lockingTestCase {
			var a = fakeJob("a", []JobSpecifier{})
			var b = fakeJob("b", []JobSpecifier{})
			var c = fakeJob("c", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, b, c},
				orderedJobs: []Job{a, b, c},
			}
		}),

		Entry("multiple jobs, single dependency", func() lockingTestCase {
			var a = fakeJob("a", []JobSpecifier{})
			var b = fakeJob("b", []JobSpecifier{{Name: "c"}})
			var c = fakeJob("c", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, c, b},
				orderedJobs: []Job{a, b, c},
			}
		}),

		Entry("multiple jobs, dependency on non-existent job", func() lockingTestCase {
			var a = fakeJob("a", []JobSpecifier{})
			var b = fakeJob("b", []JobSpecifier{{Name: "e"}})
			var c = fakeJob("c", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, b, c},
				orderedJobs: []Job{a, b, c},
			}
		}),

		Entry("multiple jobs, double dependency", func() lockingTestCase {
			var a = fakeJob("a", []JobSpecifier{})
			var b = fakeJob("b", []JobSpecifier{{Name: "c"}, {Name: "d"}})
			var c = fakeJob("c", []JobSpecifier{})
			var d = fakeJob("d", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, c, d, b},
				orderedJobs: []Job{a, b, c, d},
			}
		}),

		Entry("multiple jobs, chain of dependencies", func() lockingTestCase {
			var a = fakeJob("a", []JobSpecifier{{Name: "b"}})
			var b = fakeJob("b", []JobSpecifier{{Name: "c"}})
			var c = fakeJob("c", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{c, b, a},
				orderedJobs: []Job{a, b, c},
			}
		}),

		Entry("multiple jobs, multiple instances of the same dependee", func() lockingTestCase {
			var a = fakeJob("a", []JobSpecifier{})
			var b = fakeJob("b", []JobSpecifier{{Name: "c"}})
			var c1 = fakeJobOnInstance("c", "instance_group/0", []JobSpecifier{})
			var c2 = fakeJobOnInstance("c", "instance_group/1", []JobSpecifier{})
			var c3 = fakeJobOnInstance("c", "instance_group/2", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{c1, c2, c3, a, b},
				orderedJobs: []Job{a, b, c1, c2, c3},
			}
		}),

		Entry("multiple jobs, multiple instances of the same dependent", func() lockingTestCase {
			var a = fakeJob("a", []JobSpecifier{})
			var b1 = fakeJobOnInstance("b", "instance_group/0", []JobSpecifier{{Name: "c"}})
			var b2 = fakeJobOnInstance("b", "instance_group/1", []JobSpecifier{{Name: "c"}})
			var b3 = fakeJobOnInstance("b", "instance_group/2", []JobSpecifier{{Name: "c"}})
			var c = fakeJob("c", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, c, b1, b2, b3},
				orderedJobs: []Job{a, b1, b2, b3, c},
			}
		}),
	)
})

func fakeJob(name string, shouldBeLockedBefore []JobSpecifier) *fakes.FakeJob {
	instanceIdentifier := strconv.FormatInt(time.Now().UnixNano(), 16)
	return fakeJobOnInstance(name, instanceIdentifier, shouldBeLockedBefore)
}

func fakeJobOnInstance(name string, instanceIdentifier string, shouldBeLockedBefore []JobSpecifier) *fakes.FakeJob {
	job := new(fakes.FakeJob)
	job.NameReturns(name)
	job.InstanceIdentifierReturns(instanceIdentifier)
	job.ShouldBeLockedBeforeReturns(shouldBeLockedBefore)
	return job
}
