package orchestrator_test

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strconv"
	"time"

	orchestratorFakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("Reverse", func() {
	var job1 *orchestratorFakes.FakeJob
	var job2 *orchestratorFakes.FakeJob
	var job3 *orchestratorFakes.FakeJob
	var job4 *orchestratorFakes.FakeJob
	var incomingJobs [][]orchestrator.Job

	BeforeEach(func() {
		job1 = fakeJob()
		job2 = fakeJob()
		job3 = fakeJob()
		job4 = fakeJob()

		incomingJobs = [][]orchestrator.Job{
			{job1}, {job2, job3}, {job4},
		}
	})

	It("returns the list of jobs in reverse order", func() {
		reversedJobs := orchestrator.Reverse(incomingJobs)
		for i := 0; i < 3; i++ {
			reversedJobElement := reversedJobs[2-i]
			actualJobElement := incomingJobs[i]
			Expect(len(actualJobElement)).To(Equal(len(reversedJobElement)))
			for index, actualJob := range actualJobElement {
				Expect(actualJob.InstanceIdentifier()).To(Equal(reversedJobElement[index].InstanceIdentifier()), "list of jobs was not reversed")
			}
		}
	})
})

func fakeJob() *orchestratorFakes.FakeJob {
	instanceIdentifier := strconv.FormatInt(time.Now().UnixNano(), 16)
	job := new(orchestratorFakes.FakeJob)
	job.InstanceIdentifierReturns(instanceIdentifier)
	return job
}
