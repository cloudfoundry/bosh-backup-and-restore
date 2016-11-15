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
		deployment        *fakes.FakeDeployment
		deploymentManager *fakes.FakeDeploymentManager
		instances         []backuper.Instance
		artifact          *fakes.FakeArtifact
		artifactCreator   *fakes.FakeArtifactCreator
		logger            *fakes.FakeLogger
		deploymentName    = "foobarbaz"
		actualBackupError error
		backupWriter      *fakes.FakeWriteCloser
	)

	BeforeEach(func() {
		deployment = new(fakes.FakeDeployment)
		deploymentManager = new(fakes.FakeDeploymentManager)
		boshDirector = new(fakes.FakeBoshDirector)
		artifactCreator = new(fakes.FakeArtifactCreator)
		artifact = new(fakes.FakeArtifact)
		logger = new(fakes.FakeLogger)
		instance = new(fakes.FakeInstance)
		instances = []backuper.Instance{instance}
		backupWriter = new(fakes.FakeWriteCloser)
		b = backuper.New(boshDirector, artifactCreator.Spy, logger, deploymentManager)
	})

	JustBeforeEach(func() {
		actualBackupError = b.Backup(deploymentName)
	})

	Context("backups up an instance", func() {
		BeforeEach(func() {
			artifactCreator.Returns(artifact, nil)
			deploymentManager.FindReturns(deployment, nil)
			deployment.IsBackupableReturns(true, nil)
			deployment.CleanupReturns(nil)
			artifact.CreateFileReturns(backupWriter, nil)
			deployment.CopyRemoteBackupsToLocalArtifactReturns(nil)
		})

		It("does not fail", func() {
			Expect(actualBackupError).ToNot(HaveOccurred())
		})

		It("finds the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the deployment is backupable", func() {
			Expect(deployment.IsBackupableCallCount()).To(Equal(1))
		})

		It("runs backup scripts on the deployment", func() {
			Expect(deployment.BackupCallCount()).To(Equal(1))
		})

		It("ensures that deployment is cleaned up", func() {
			Expect(deployment.CleanupCallCount()).To(Equal(1))
		})

		It("creates a local artifact", func() {
			Expect(artifactCreator.CallCount()).To(Equal(1))
		})

		It("names the artifact after the deployment", func() {
			Expect(artifactCreator.ArgsForCall(0)).To(Equal(deploymentName))
		})

		It("drains the backup to the artifact", func() {
			Expect(deployment.CopyRemoteBackupsToLocalArtifactCallCount()).To(Equal(1))
			Expect(deployment.CopyRemoteBackupsToLocalArtifactArgsForCall(0)).To(Equal(artifact))
		})
	})

	Describe("failures", func() {
		var expectedError = fmt.Errorf("Jesus!")
		Context("fails to find deployment", func() {
			BeforeEach(func() {
				deploymentManager.FindReturns(nil, expectedError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(expectedError))
			})
		})

		Context("fails when checking if deployment is backupable", func() {
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(false, expectedError)
			})

			It("finds deployment with the deployment name", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
				Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(expectedError))
			})
			It("ensures that deployment is cleaned up", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("fails if the deployment is not backupable", func() {
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(false, nil)
			})

			It("finds a deployment with the deployment name", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
				Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
			})
			It("checks if the deployment is backupable", func() {
				Expect(deployment.IsBackupableCallCount()).To(Equal(1))
			})
			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError("Deployment '" + deploymentName + "' has no backup scripts"))
			})
			It("ensures that deployment is cleaned up", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("fails if backup cannot be drained", func() {
			var drainError = fmt.Errorf("they are bringing crime")
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true, nil)
				artifactCreator.Returns(artifact, nil)
				deployment.CopyRemoteBackupsToLocalArtifactReturns(drainError)
			})

			It("check if the deployment is backupable", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
				Expect(deployment.IsBackupableCallCount()).To(Equal(1))
			})

			It("backs up the deployment", func() {
				Expect(deployment.BackupCallCount()).To(Equal(1))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(drainError))
			})
		})

		Context("fails if artifact cannot be created", func() {
			var artifactError = fmt.Errorf("they are bringing crime")
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true, nil)

				artifactCreator.Returns(nil, artifactError)
			})

			It("check if the deployment is backupable", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
				Expect(deployment.IsBackupableCallCount()).To(Equal(1))
			})

			It("dosent backup the deployment", func() {
				Expect(deployment.BackupCallCount()).To(BeZero())
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(artifactError))
			})
		})

		Context("fails if backup is not a success", func() {
			var backupError = fmt.Errorf("i have the best words")
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true, nil)

				artifactCreator.Returns(artifact, nil)
				deployment.BackupReturns(backupError)
			})

			It("check if the deployment is backupable", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
				Expect(deployment.IsBackupableCallCount()).To(Equal(1))
			})

			It("does try to backup the instance", func() {
				Expect(deployment.BackupCallCount()).To(Equal(1))
			})

			It("does not try to create files in the artifact", func() {
				Expect(artifact.CreateFileCallCount()).To(BeZero())
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(backupError))
			})
		})
	})
})

var _ = Describe("restore", func() {
	Context("restores an instance from backup", func() {
		var (
			restoreError    error
			artifactCreator *fakes.FakeArtifactCreator
			artifact        *fakes.FakeArtifact
			boshDirector    *fakes.FakeBoshDirector
			logger          *fakes.FakeLogger
			instance        *fakes.FakeInstance
			instances       []backuper.Instance
			b               *backuper.Backuper
			deploymentName  string
		)

		BeforeEach(func() {
			instance = new(fakes.FakeInstance)
			instances = []backuper.Instance{instance}
			boshDirector = new(fakes.FakeBoshDirector)
			logger = new(fakes.FakeLogger)
			artifactCreator = new(fakes.FakeArtifactCreator)
			artifact = new(fakes.FakeArtifact)

			artifactCreator.Returns(artifact, nil)
			boshDirector.FindInstancesReturns(instances, nil)
			instance.IsRestorableReturns(true, nil)
			artifact.DeploymentMatchesReturns(true, nil)

			b = backuper.New(boshDirector, artifactCreator.Spy, logger, backuper.NewBoshDeploymentManager(boshDirector, logger))

			deploymentName = "deployment-to-restore"
		})

		JustBeforeEach(func() {
			restoreError = b.Restore(deploymentName)
		})

		It("does not fail", func() {
			Expect(restoreError).NotTo(HaveOccurred())
		})

		It("ensures that instance is cleaned up", func() {
			Expect(instance.CleanupCallCount()).To(Equal(1))
		})

		It("finds a instances for the deployment", func() {
			Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
			Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the instance is restorable", func() {
			Expect(instance.IsRestorableCallCount()).To(Equal(1))
		})

		It("checks that the deployment topology matches the topology of the backup", func() {
			Expect(artifactCreator.CallCount()).To(Equal(1))
			Expect(artifact.DeploymentMatchesCallCount()).To(Equal(1))

			name, instances := artifact.DeploymentMatchesArgsForCall(0)
			Expect(name).To(Equal(deploymentName))
			Expect(instances).To(ContainElement(instance))
		})

		It("streams the local backup to the instance", func() {
			Expect(instance.StreamBackupToRemoteCallCount()).To(Equal(1))
		})

		It("calls restore on the instance", func() {
			Expect(instance.RestoreCallCount()).To(Equal(1))
		})

		Describe("failures", func() {
			Context("fails to find instances", func() {
				BeforeEach(func() {
					boshDirector.FindInstancesReturns(nil, fmt.Errorf("they will pay for the wall"))
				})

				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(MatchError("they will pay for the wall"))
				})
			})

			Context("if no instances are restorable", func() {
				BeforeEach(func() {
					instance.IsRestorableReturns(false, nil)
				})

				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(HaveOccurred())
				})
			})

			Context("if checking the instance's restorable status fails", func() {
				BeforeEach(func() {
					instance.IsRestorableReturns(true, fmt.Errorf("the beauty of me is that I'm very rich"))
				})
				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(HaveOccurred())
				})
			})

			Context("if the deployment's topology doesn't match that of the backup", func() {
				BeforeEach(func() {
					artifact.DeploymentMatchesReturns(false, nil)
				})

				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(HaveOccurred())
				})
			})

			Context("if checking the deployment topology fails", func() {
				BeforeEach(func() {
					artifact.DeploymentMatchesReturns(true, fmt.Errorf("my fingers are long and beautiful"))
				})

				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(HaveOccurred())
				})
			})

			Context("if streaming the backup to the remote fails", func() {
				BeforeEach(func() {
					instance.StreamBackupToRemoteReturns(fmt.Errorf("Broken pipe"))
				})

				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError.Error()).To(Equal("Unable to send backup to remote machine. Got error: Broken pipe"))
				})
			})
		})
	})
})
