package orchestrator_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("Check", func() {
	var (
		b                        *orchestrator.BackupChecker
		deployment               *fakes.FakeDeployment
		deploymentManager        *fakes.FakeDeploymentManager
		logger                   *fakes.FakeLogger
		lockOrderer              *fakes.FakeLockOrderer
		deploymentName           = "foobarbaz"
		actualCanBeBackedUpError error
	)

	BeforeEach(func() {
		deployment = new(fakes.FakeDeployment)
		deploymentManager = new(fakes.FakeDeploymentManager)
		logger = new(fakes.FakeLogger)
		b = orchestrator.NewBackupChecker(logger, deploymentManager, lockOrderer)
	})

	JustBeforeEach(func() {
		actualCanBeBackedUpError = b.Check(deploymentName)
	})

	Context("when the deployment can be backed up", func() {
		BeforeEach(func() {
			deploymentManager.FindReturns(deployment, nil)
			deployment.IsBackupableReturns(true)
			deployment.CleanupReturns(nil)
		})

		It("succeeds", func() {
			Expect(actualCanBeBackedUpError).NotTo(HaveOccurred())
		})

		It("finds the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the deployment is backupable", func() {
			Expect(deployment.IsBackupableCallCount()).To(Equal(1))
		})

		It("shouldn't do a backup", func() {
			Expect(deployment.BackupCallCount()).To(Equal(0))
		})

		It("ensures that deployment is cleaned up", func() {
			Expect(deployment.CleanupCallCount()).To(Equal(1))
		})
	})

	Context("when the deployment doesn't exist", func() {
		BeforeEach(func() {
			deploymentManager.FindReturns(nil, fmt.Errorf("deployment not found"))
			deployment.IsBackupableReturns(true)
			deployment.CleanupReturns(nil)
		})

		It("returns an error", func() {
			Expect(actualCanBeBackedUpError).To(HaveOccurred())
		})

		It("attempts to find the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})
	})
})
