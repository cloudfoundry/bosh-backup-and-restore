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
		inputJobs                []Job
		orderedJobs              [][]Job
		errorMessage             string
		orderConstraintSpecifier orderConstraintSpecifier
	}

	DescribeTable("KahnLockOrderer",
		func(testCaseBuilder func() lockingTestCase) {
			testCase := testCaseBuilder()
			lockOrderer := newKahnLockOrderer(testCase.orderConstraintSpecifier)

			orderedJobs, err := lockOrderer.Order(testCase.inputJobs)
			Expect(orderedJobs).To(Equal(testCase.orderedJobs))
			if testCase.errorMessage == "" {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(MatchError(ContainSubstring(testCase.errorMessage)))
			}
		},

		Entry("no jobs", func() lockingTestCase {
			return lockingTestCase{
				inputJobs:   []Job{},
				orderedJobs: [][]Job{},
			}
		}),

		Entry("one job", func() lockingTestCase {
			var job = fakeJob("test", "releasetest")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()

			return lockingTestCase{
				inputJobs:                []Job{job},
				orderedJobs:              [][]Job{{job}},
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),

		Entry("multiple jobs, no dependencies", func() lockingTestCase {
			var a = fakeJob("a", "releasea")
			var b = fakeJob("b", "releaseb")
			var c = fakeJob("c", "releasec")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()

			return lockingTestCase{
				inputJobs:                []Job{a, b, c},
				orderedJobs:              [][]Job{{a, b, c}},
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),

		Entry("multiple jobs, single dependency", func() lockingTestCase {
			var a = fakeJob("a", "releasea")
			var b = fakeJob("b", "releaseb")
			var c = fakeJob("c", "releasec")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()
			orderConstraintSpecifier.AddConstraint(b, []JobSpecifier{{Name: "c", Release: "releasec"}})

			return lockingTestCase{
				inputJobs:                []Job{a, c, b},
				orderedJobs:              [][]Job{{a, b}, {c}},
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),

		Entry("multiple jobs, dependency on non-existent job", func() lockingTestCase {
			var a = fakeJob("a", "releasea")
			var b = fakeJob("b", "releaseb")
			var c = fakeJob("c", "releasec")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()
			orderConstraintSpecifier.AddConstraint(b, []JobSpecifier{{Name: "e", Release: "releasee"}})

			return lockingTestCase{
				inputJobs:                []Job{a, b, c},
				orderedJobs:              [][]Job{{a, b, c}},
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),

		Entry("multiple jobs, double dependency", func() lockingTestCase {
			var a = fakeJob("a", "releasea")
			var b = fakeJob("b", "releaseb")
			var c = fakeJob("c", "releasec")
			var d = fakeJob("d", "released")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()
			orderConstraintSpecifier.AddConstraint(b, []JobSpecifier{{Name: "c", Release: "releasec"}, {Name: "d", Release: "released"}})

			return lockingTestCase{
				inputJobs:                []Job{a, c, d, b},
				orderedJobs:              [][]Job{{a, b}, {c, d}},
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),

		Entry("multiple jobs, chain of dependencies", func() lockingTestCase {
			var a = fakeJob("a", "releasea")
			var b = fakeJob("b", "releaseb")
			var c = fakeJob("c", "releasec")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()
			orderConstraintSpecifier.AddConstraint(a, []JobSpecifier{{Name: "b", Release: "releaseb"}})
			orderConstraintSpecifier.AddConstraint(b, []JobSpecifier{{Name: "c", Release: "releasec"}})

			return lockingTestCase{
				inputJobs:                []Job{c, b, a},
				orderedJobs:              [][]Job{{a}, {b}, {c}},
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),

		Entry("multiple instances of the same job that comes after", func() lockingTestCase {
			var a = fakeJob("a", "releasea")
			var b = fakeJob("b", "releaseb")
			var c1 = fakeJobOnInstance("c", "releasec", "instance_group/0")
			var c2 = fakeJobOnInstance("c", "releasec", "instance_group/1")
			var c3 = fakeJobOnInstance("c", "releasec", "instance_group/2")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()
			orderConstraintSpecifier.AddConstraint(b, []JobSpecifier{{Name: "c", Release: "releasec"}})

			return lockingTestCase{
				inputJobs:                []Job{c1, c2, c3, a, b},
				orderedJobs:              [][]Job{{a, b}, {c1, c2, c3}},
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),

		Entry("multiple instances of the job that comes before", func() lockingTestCase {
			var a = fakeJob("a", "releasea")
			var b1 = fakeJobOnInstance("b", "releaseb", "instance_group/0")
			var b2 = fakeJobOnInstance("b", "releaseb", "instance_group/1")
			var b3 = fakeJobOnInstance("b", "releaseb", "instance_group/2")
			var c = fakeJob("c", "releasec")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()
			orderConstraintSpecifier.AddConstraint(b1, []JobSpecifier{{Name: "c", Release: "releasec"}})
			orderConstraintSpecifier.AddConstraint(b2, []JobSpecifier{{Name: "c", Release: "releasec"}})
			orderConstraintSpecifier.AddConstraint(b3, []JobSpecifier{{Name: "c", Release: "releasec"}})

			return lockingTestCase{
				inputJobs:                []Job{a, c, b1, b2, b3},
				orderedJobs:              [][]Job{{a, b1, b2, b3}, {c}},
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),

		Entry("multiple jobs from different releases, multiple instances of the same dependent", func() lockingTestCase {
			var a = fakeJobOnInstance("a", "releasea", "instance_group/0")
			var b = fakeJobOnInstance("b", "releaseb", "instance_group/1")
			var c1 = fakeJobOnInstance("c", "release1", "instance_group/1")
			var c2 = fakeJob("c", "release2")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()
			orderConstraintSpecifier.AddConstraint(a, []JobSpecifier{{Name: "c", Release: "release1"}})
			orderConstraintSpecifier.AddConstraint(b, []JobSpecifier{{Name: "c", Release: "release2"}})
			orderConstraintSpecifier.AddConstraint(c1, []JobSpecifier{{Name: "c", Release: "release2"}})

			return lockingTestCase{
				inputJobs:                []Job{a, c1, b, c2},
				orderedJobs:              [][]Job{{a, b}, {c1}, {c2}},
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),

		Entry("multiple jobs with cyclic dependencies", func() lockingTestCase {
			var a = fakeJobOnInstance("a", "releasea", "instance_group/0")
			var b = fakeJobOnInstance("b", "releaseb", "instance_group/1")
			var c = fakeJobOnInstance("c", "releasec", "instance_group/2")

			orderConstraintSpecifier := NewFakeOrderConstraintSpecifier()
			orderConstraintSpecifier.AddConstraint(a, []JobSpecifier{{Name: "c", Release: "releasec"}})
			orderConstraintSpecifier.AddConstraint(b, []JobSpecifier{{Name: "a", Release: "releasea"}})
			orderConstraintSpecifier.AddConstraint(c, []JobSpecifier{{Name: "b", Release: "releaseb"}})

			return lockingTestCase{
				inputJobs:                []Job{a, b, c},
				errorMessage:             "job locking dependency graph is cyclic",
				orderConstraintSpecifier: orderConstraintSpecifier,
			}
		}),
	)

	Describe("NewKahnBackupLockOrderer", func() {
		It("creates a kahn backup lock order with the backup lock constraint", func() {
			Expect(NewKahnBackupLockOrderer()).To(Equal(newKahnLockOrderer(NewBackupOrderConstraintSpecifier())))
		})
	})

	Describe("NewKahnRestoreLockOrderer", func() {
		It("creates a kahn restore lock order with the restore lock constraint", func() {
			Expect(NewKahnRestoreLockOrderer()).To(Equal(newKahnLockOrderer(NewRestoreOrderConstraintSpecifier())))
		})
	})
})

func NewFakeOrderConstraintSpecifier() *FakeOrderConstraintSpecifier {
	return &FakeOrderConstraintSpecifier{mapping: map[Job][]JobSpecifier{}}
}

func fakeJob(name string, release string) *fakes.FakeJob {
	instanceIdentifier := strconv.FormatInt(time.Now().UnixNano(), 16)
	return fakeJobOnInstance(name, release, instanceIdentifier)
}

func fakeJobOnInstance(name, release, instanceIdentifier string) *fakes.FakeJob {
	job := new(fakes.FakeJob)
	job.NameReturns(name)
	job.ReleaseReturns(release)
	job.InstanceIdentifierReturns(instanceIdentifier)
	return job

}

type FakeOrderConstraintSpecifier struct {
	mapping map[Job][]JobSpecifier
}

func (f *FakeOrderConstraintSpecifier) AddConstraint(job Job, specification []JobSpecifier) {
	f.mapping[job] = specification
}

func (f *FakeOrderConstraintSpecifier) Before(job Job) []JobSpecifier {
	return f.mapping[job]
}
