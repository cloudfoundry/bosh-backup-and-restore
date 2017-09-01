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
			var job = fakeJob("test", "releasetest", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{job},
				orderedJobs: []Job{job},
			}
		}),

		Entry("one job, dependency on non-existent job", func() lockingTestCase {
			var job = fakeJob("test", "releasetest", []JobSpecifier{{Name: "non-existent"}})

			return lockingTestCase{
				inputJobs:   []Job{job},
				orderedJobs: []Job{job},
			}
		}),

		Entry("multiple jobs, no dependencies", func() lockingTestCase {
			var a = fakeJob("a", "releasea", []JobSpecifier{})
			var b = fakeJob("b", "releaseb", []JobSpecifier{})
			var c = fakeJob("c", "releasec", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, b, c},
				orderedJobs: []Job{a, b, c},
			}
		}),

		Entry("multiple jobs, single dependency", func() lockingTestCase {
			var a = fakeJob("a", "releasea", []JobSpecifier{})
			var b = fakeJob("b", "releaseb", []JobSpecifier{{Name: "c", Release: "releasec"}})
			var c = fakeJob("c", "releasec", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, c, b},
				orderedJobs: []Job{a, b, c},
			}
		}),

		Entry("multiple jobs, dependency on non-existent job", func() lockingTestCase {
			var a = fakeJob("a", "releasea", []JobSpecifier{})
			var b = fakeJob("b", "releaseb", []JobSpecifier{{Name: "e", Release: "releasee"}})
			var c = fakeJob("c", "releasec", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, b, c},
				orderedJobs: []Job{a, b, c},
			}
		}),

		Entry("multiple jobs, double dependency", func() lockingTestCase {
			var a = fakeJob("a", "releasea", []JobSpecifier{})
			var b = fakeJob("b", "releaseb", []JobSpecifier{{Name: "c", Release: "releasec"}, {Name: "d", Release: "released"}})
			var c = fakeJob("c", "releasec", []JobSpecifier{})
			var d = fakeJob("d", "released", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, c, d, b},
				orderedJobs: []Job{a, b, c, d},
			}
		}),

		Entry("multiple jobs, chain of dependencies", func() lockingTestCase {
			var a = fakeJob("a", "releasea", []JobSpecifier{{Name: "b", Release: "releaseb"}})
			var b = fakeJob("b", "releaseb", []JobSpecifier{{Name: "c", Release: "releasec"}})
			var c = fakeJob("c", "releasec", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{c, b, a},
				orderedJobs: []Job{a, b, c},
			}
		}),

		Entry("multiple jobs, multiple instances of the same dependee", func() lockingTestCase {
			var a = fakeJob("a", "releasea", []JobSpecifier{})
			var b = fakeJob("b", "releaseb", []JobSpecifier{{Name: "c", Release: "releasec"}})
			var c1 = fakeJobOnInstance("c", "releasec", "instance_group/0", []JobSpecifier{})
			var c2 = fakeJobOnInstance("c", "releasec", "instance_group/1", []JobSpecifier{})
			var c3 = fakeJobOnInstance("c", "releasec", "instance_group/2", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{c1, c2, c3, a, b},
				orderedJobs: []Job{a, b, c1, c2, c3},
			}
		}),

		Entry("multiple jobs, multiple instances of the same dependent", func() lockingTestCase {
			var a = fakeJob("a", "releasea", []JobSpecifier{})
			var b1 = fakeJobOnInstance("b", "releaseb", "instance_group/0", []JobSpecifier{{Name: "c", Release: "releasec"}})
			var b2 = fakeJobOnInstance("b", "releaseb", "instance_group/1", []JobSpecifier{{Name: "c", Release: "releasec"}})
			var b3 = fakeJobOnInstance("b", "releaseb", "instance_group/2", []JobSpecifier{{Name: "c", Release: "releasec"}})
			var c = fakeJob("c", "releasec", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, c, b1, b2, b3},
				orderedJobs: []Job{a, b1, b2, b3, c},
			}
		}),

		Entry("multiple jobs from different releases, multiple instances of the same dependent", func() lockingTestCase {
			var a = fakeJobOnInstance("a", "releasea", "instance_group/0", []JobSpecifier{{Name: "c", Release: "release1"}})
			var b = fakeJobOnInstance("b", "releaseb", "instance_group/1", []JobSpecifier{{Name: "c", Release: "release2"}})
			var c1 = fakeJobOnInstance("c", "release1", "instance_group/1", []JobSpecifier{{Name: "c", Release: "release2"}})
			var c2 = fakeJob("c", "release2", []JobSpecifier{})

			return lockingTestCase{
				inputJobs:   []Job{a, c1, b, c2},
				orderedJobs: []Job{a, c1, b, c2},
			}
		}),
	)
})

func fakeJob(name string, release string, shouldBeLockedBefore []JobSpecifier) *fakes.FakeJob {
	instanceIdentifier := strconv.FormatInt(time.Now().UnixNano(), 16)
	return fakeJobOnInstance(name, release, instanceIdentifier, shouldBeLockedBefore)
}

func fakeJobOnInstance(name, release, instanceIdentifier string, shouldBeLockedBefore []JobSpecifier) *fakes.FakeJob {
	job := new(fakes.FakeJob)
	job.NameReturns(name)
	job.ReleaseReturns(release)
	job.InstanceIdentifierReturns(instanceIdentifier)
	job.ShouldBeLockedBeforeReturns(shouldBeLockedBefore)
	return job
}
