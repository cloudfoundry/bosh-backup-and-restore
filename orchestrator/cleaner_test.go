package orchestrator_test

import (
	"fmt"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cleanup", func() {
	var (
		c                 *orchestrator.Cleaner
		deployment        *fakes.FakeDeployment
		deploymentManager *fakes.FakeDeploymentManager
		deploymentName    = "foobarbaz"
		cleanupError      error
		logger            *fakes.FakeLogger
	)

	BeforeEach(func() {
		deployment = new(fakes.FakeDeployment)
		deploymentManager = new(fakes.FakeDeploymentManager)
		logger = new(fakes.FakeLogger)
		c = orchestrator.NewCleaner(logger, deploymentManager)
	})

	JustBeforeEach(func() {
		cleanupError = c.Cleanup(deploymentName)
	})

	Context("when the deployment can be cleaned up", func() {
		BeforeEach(func() {
			deploymentManager.FindReturns(deployment, nil)
			deployment.CleanupPreviousReturns(nil)
		})

		It("finds the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})

		It("ensures that deployment is cleaned up", func() {
			Expect(deployment.CleanupPreviousCallCount()).To(Equal(1))
		})

		It("ensures that deployment is unlocked", func() {
			Expect(deployment.PostBackupUnlockCallCount()).To(Equal(1))
		})
	})

	Context("when the deployment doesn't exist", func() {
		BeforeEach(func() {
			deploymentManager.FindReturns(nil, fmt.Errorf("deployment not found"))
			deployment.CleanupPreviousReturns(nil)
		})

		It("attempts to find the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})

		It("fails", func() {
			Expect(cleanupError).To(HaveOccurred())
		})
	})

	Context("when the cleanup fails", func() {
		var deploymentCleanupError error

		BeforeEach(func() {
			deploymentCleanupError = fmt.Errorf("cleanup error")
			deploymentManager.FindReturns(deployment, nil)
			deployment.CleanupPreviousReturns(deploymentCleanupError)
		})

		It("returns an error", func() {
			Expect(cleanupError.Error()).To(ContainSubstring(deploymentCleanupError.Error()))
		})

		It("continues with unlock", func() {
			Expect(deployment.PostBackupUnlockCallCount()).To(Equal(1))
		})
	})

	Context("when the unlocking fails", func() {
		var instanceUnlockError error

		BeforeEach(func() {
			instanceUnlockError = fmt.Errorf("unlock error")
			deploymentManager.FindReturns(deployment, nil)
			deployment.CleanupPreviousReturns(nil)
			deployment.PostBackupUnlockReturns(instanceUnlockError)
		})

		It("ensures that deployment is cleaned up", func() {
			Expect(deployment.CleanupPreviousCallCount()).To(Equal(1))
		})

		It("returns an error", func() {
			Expect(cleanupError.Error()).To(ContainSubstring(instanceUnlockError.Error()))
		})
	})

	Context("when cleanup and unlocking fails", func() {
		var instanceUnlockError error
		var instanceCleanupError error

		BeforeEach(func() {
			instanceUnlockError = fmt.Errorf("unlock error")
			instanceCleanupError = fmt.Errorf("cleanup error")
			deploymentManager.FindReturns(deployment, nil)
			deployment.CleanupPreviousReturns(instanceCleanupError)
			deployment.PostBackupUnlockReturns(instanceUnlockError)
		})

		It("returns both errors", func() {
			Expect(cleanupError.Error()).To(ContainSubstring(instanceUnlockError.Error()))
			Expect(cleanupError.Error()).To(ContainSubstring(instanceCleanupError.Error()))
		})
	})
})
