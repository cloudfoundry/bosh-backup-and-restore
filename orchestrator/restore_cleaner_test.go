package orchestrator_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Restore Cleanup", func() {
	var (
		restoreCleaner    *orchestrator.RestoreCleaner
		deployment        *fakes.FakeDeployment
		deploymentManager *fakes.FakeDeploymentManager
		deploymentName    = "foobarbaz"
		cleanupError      error
		logger            *fakes.FakeLogger
		fakeLockOrderer   *fakes.FakeLockOrderer
	)

	BeforeEach(func() {
		deployment = new(fakes.FakeDeployment)
		deploymentManager = new(fakes.FakeDeploymentManager)
		logger = new(fakes.FakeLogger)
		restoreCleaner = orchestrator.NewRestoreCleaner(logger, deploymentManager, fakeLockOrderer, executor.NewSerialExecutor())
	})

	JustBeforeEach(func() {
		cleanupError = restoreCleaner.Cleanup(deploymentName)
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
			Expect(deployment.PostRestoreUnlockCallCount()).To(Equal(1))
		})
	})

	Context("the deployment actions are in order", func() {
		var currentSequenceNumber, unlockCallIndex, cleanupCallIndex int
		BeforeEach(func() {
			deploymentManager.FindReturns(deployment, nil)
			deployment.PostRestoreUnlockStub = func(orderer orchestrator.LockOrderer, _ executor.Executor) error {
				unlockCallIndex = currentSequenceNumber
				currentSequenceNumber = currentSequenceNumber + 1
				return nil
			}
			deployment.CleanupPreviousStub = func() error {
				cleanupCallIndex = currentSequenceNumber
				currentSequenceNumber = currentSequenceNumber + 1
				return nil
			}
		})

		It("unlocks and then cleanups", func() {
			Expect(unlockCallIndex).To(Equal(0))
			Expect(cleanupCallIndex).To(Equal(1))
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
			Expect(cleanupError).To(MatchError(ContainSubstring("deployment not found")))
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
			Expect(deployment.PostRestoreUnlockCallCount()).To(Equal(1))
		})
	})

	Context("when the unlocking fails", func() {
		var instanceUnlockError error

		BeforeEach(func() {
			instanceUnlockError = fmt.Errorf("unlock error")
			deploymentManager.FindReturns(deployment, nil)
			deployment.CleanupPreviousReturns(nil)
			deployment.PostRestoreUnlockReturns(instanceUnlockError)
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
			deployment.PostRestoreUnlockReturns(instanceUnlockError)
		})

		It("returns both errors", func() {
			Expect(cleanupError.Error()).To(ContainSubstring(instanceUnlockError.Error()))
			Expect(cleanupError.Error()).To(ContainSubstring(instanceCleanupError.Error()))
		})
	})
})
