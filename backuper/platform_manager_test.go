package backuper_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"
)

var _ = Describe("PlatformManager", func() {
	var platformManager backuper.PlatformManager
	var deploymentManager *fakes.FakeDeploymentManager
	var logger *fakes.FakeLogger
	var query string

	var actualErr error
	var actualPlatform backuper.Platform

	BeforeEach(func() {
		logger = new(fakes.FakeLogger)
		deploymentManager = new(fakes.FakeDeploymentManager)
		platformManager = backuper.NewBoshPlatformManager(deploymentManager, logger)
	})
	Context("Find", func() {
		JustBeforeEach(func() {
			actualPlatform, actualErr = platformManager.Find(query)
		})
		Context("single deployment", func() {
			BeforeEach(func() {
				query = "only_one_deployment"
			})
			It("finds deployments using the deployments manager", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
				Expect(deploymentManager.FindArgsForCall(0)).To(Equal("only_one_deployment"))
			})

			Context("errors", func() {
				var expectedError = fmt.Errorf("i am the greatest")
				BeforeEach(func() {
					deploymentManager.FindReturns(nil, expectedError)
				})
				It("fails if there is an error finding a deployment", func() {
					Expect(actualErr).To(MatchError(expectedError))
				})
			})
		})
		Context("multiple deployments", func() {
			BeforeEach(func() {
				query = "multiple_deployments,seperated_by_comma"
			})

			It("splits multiple deployments by comma and finds deployments using the deployments manager", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(2))
				Expect(deploymentManager.FindArgsForCall(0)).To(Equal("multiple_deployments"))
				Expect(deploymentManager.FindArgsForCall(1)).To(Equal("seperated_by_comma"))
			})
			Context("errors", func() {
				var expectedError = fmt.Errorf("so unfair")
				BeforeEach(func() {
					var counter = 0
					deploymentManager.FindStub = func(deploymentName string) (backuper.Deployment, error) {
						if counter == 0 {
							counter++
							return new(fakes.FakeDeployment), nil
						} else {
							return nil, expectedError
						}
					}
				})
				It("fails if there is an error finding a deployment", func() {
					Expect(actualErr).To(MatchError(expectedError))
				})
			})
		})

	})
})
