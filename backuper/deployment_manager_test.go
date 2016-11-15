package backuper_test

import (
	"fmt"

	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"
)

var _ = Describe("DeploymentManager", func() {
	var boshDirector *fakes.FakeBoshDirector
	var logger *fakes.FakeLogger
	var deploymentName = "brownie"

	var deploymentManager backuper.DeploymentManager
	BeforeEach(func() {
		boshDirector = new(fakes.FakeBoshDirector)
		logger = new(fakes.FakeLogger)
	})
	JustBeforeEach(func() {
		deploymentManager = backuper.NewBoshDeploymentManager(boshDirector, logger)
	})

	Context("Find", func() {
		var findError error
		var deployment backuper.Deployment
		var instances []backuper.Instance
		BeforeEach(func() {
			instances = []backuper.Instance{new(fakes.FakeInstance)}
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
			Expect(deployment).To(Equal(backuper.NewBoshDeployment(boshDirector, logger, instances)))
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
