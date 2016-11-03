package backuper_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"
)

var _ = Describe("Backuper", func() {
	var (
		boshDirector      *fakes.FakeBoshDirector
		b                 *backuper.Backuper
		instance          *fakes.FakeInstance
		instances         backuper.Instances
		artifactCreator   *fakes.FakeArtifactCreator
		deploymentName    = "foobarbaz"
		actualBackupError error
	)

	BeforeEach(func() {
		boshDirector = new(fakes.FakeBoshDirector)
		artifactCreator = new(fakes.FakeArtifactCreator)
		instance = new(fakes.FakeInstance)
		instances = backuper.Instances{instance}
		b = backuper.New(boshDirector, artifactCreator.Spy)
	})
	JustBeforeEach(func() {
		actualBackupError = b.Backup(deploymentName)
	})

	Context("backups up instances", func() {
		BeforeEach(func() {
			boshDirector.FindInstancesReturns(instances, nil)
			instance.IsBackupableReturns(true, nil)
			instance.CleanupReturns(nil)
		})

		It("does not fail", func() {
			Expect(actualBackupError).ToNot(HaveOccurred())
		})

		It("finds a instances for the deployment", func() {
			Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
			Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the instance is backupable", func() {
			Expect(instance.IsBackupableCallCount()).To(Equal(1))
		})

		It("runs backup scripts on the instance", func() {
			Expect(instance.BackupCallCount()).To(Equal(1))
		})

		It("ensures that instance is cleaned up", func() {
			Expect(instance.CleanupCallCount()).To(Equal(1))
		})

		It("creates a local artifact", func() {
			Expect(artifactCreator.CallCount()).To(Equal(1))
		})

		It("names the artifact after the deployment", func() {
			Expect(artifactCreator.ArgsForCall(0)).To(Equal(deploymentName))
		})
	})

	Describe("failures", func() {
		var expectedError = fmt.Errorf("Jesus!")
		Context("fails to find instances", func() {
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(nil, expectedError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(expectedError))
			})
		})

		Context("fails when checking if instances are backupable", func() {
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(false, expectedError)
			})

			It("finds instances with the deployment name", func() {
				Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
				Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(expectedError))
			})
			It("ensures that deployment is cleaned up", func() {
				Expect(instance.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("fails if the deployment is not backupable", func() {
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(false, nil)
			})

			It("finds a instances with the deployment name", func() {
				Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
				Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
			})
			It("checks if the deployment is backupable", func() {
				Expect(instance.IsBackupableCallCount()).To(Equal(1))
			})
			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError("Deployment '" + deploymentName + "' has no backup scripts"))
			})
			It("ensures that deployment is cleaned up", func() {
				Expect(instance.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("fails if artifact cannot be created", func() {
			var artifactError = fmt.Errorf("they are bringing crime")
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(true, nil)

				artifactCreator.Returns(nil, artifactError)
			})

			It("check if the deployment is backupable", func() {
				Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
				Expect(instance.IsBackupableCallCount()).To(Equal(1))
			})

			It("dosent backup the instance", func() {
				Expect(instance.BackupCallCount()).To(BeZero())
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(artifactError))
			})
		})
	})
})
