package orchestrator_test

import (
	"fmt"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("DeploymentManager", func() {
	var boshDirector *fakes.FakeBoshDirector
	var logger *fakes.FakeLogger
	var deploymentName = "brownie"

	var deploymentManager orchestrator.DeploymentManager
	BeforeEach(func() {
		boshDirector = new(fakes.FakeBoshDirector)
		logger = new(fakes.FakeLogger)
	})
	JustBeforeEach(func() {
		deploymentManager = orchestrator.NewBoshDeploymentManager(boshDirector, logger)
	})

	Context("Find", func() {
		var findError error
		var deployment orchestrator.Deployment
		var instances []orchestrator.Instance
		BeforeEach(func() {
			instances = []orchestrator.Instance{new(fakes.FakeInstance)}
			boshDirector.FindInstancesReturns(instances, nil)
		})
		JustBeforeEach(func() {
			deployment, findError = deploymentManager.Find(deploymentName)
		})
		It("asks the bosh director for instances", func() {
			Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
			Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
		})
		It("returns the deployment manager with instances", func() {
			Expect(deployment).To(Equal(orchestrator.NewBoshDeployment(logger, instances)))
		})

		Context("error finding instances", func() {
			var expectedFindError = fmt.Errorf("some I assume are good people")
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(nil, expectedFindError)
			})

			It("returns an error", func() {
				Expect(findError).To(MatchError(expectedFindError))
			})
		})

	})

})
