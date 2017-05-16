package orchestrator_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("Backup", func() {
	var (
		b                 *orchestrator.Backuper
		deployment        *fakes.FakeDeployment
		deploymentManager *fakes.FakeDeploymentManager
		artifact          *fakes.FakeArtifact
		artifactManager   *fakes.FakeArtifactManager
		logger            *fakes.FakeLogger
		deploymentName    = "foobarbaz"
		actualBackupError orchestrator.Error
	)

	BeforeEach(func() {
		deployment = new(fakes.FakeDeployment)
		deploymentManager = new(fakes.FakeDeploymentManager)
		artifactManager = new(fakes.FakeArtifactManager)
		artifact = new(fakes.FakeArtifact)
		logger = new(fakes.FakeLogger)
		b = orchestrator.NewBackuper(artifactManager, logger, deploymentManager)
	})

	JustBeforeEach(func() {
		actualBackupError = b.Backup(deploymentName)
	})

	Context("backs up a deployment", func() {
		BeforeEach(func() {
			artifactManager.CreateReturns(artifact, nil)
			artifactManager.ExistsReturns(false)
			deploymentManager.FindReturns(deployment, nil)
			deployment.IsBackupableReturns(true)
			deployment.HasValidBackupMetadataReturns(true)
			deployment.CleanupReturns(nil)
			deployment.CopyRemoteBackupToLocalReturns(nil)
		})

		It("does not fail", func() {
			Expect(actualBackupError).NotTo(HaveOccurred())
		})

		It("finds the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})

		It("saves the deployment manifest", func() {
			Expect(deploymentManager.SaveManifestCallCount()).To(Equal(1))
			actualDeploymentName, actualArtifact := deploymentManager.SaveManifestArgsForCall(0)
			Expect(actualDeploymentName).To(Equal(deploymentName))
			Expect(actualArtifact).To(Equal(artifact))
		})

		It("checks if the artifact already exists", func() {
			Expect(artifactManager.ExistsCallCount()).To(Equal(1))
		})

		It("checks if the deployment is backupable", func() {
			Expect(deployment.IsBackupableCallCount()).To(Equal(1))
		})

		It("runs pre-backup-lock scripts on the deployment", func() {
			Expect(deployment.PreBackupLockCallCount()).To(Equal(1))
		})

		It("runs backup scripts on the deployment", func() {
			Expect(deployment.BackupCallCount()).To(Equal(1))
		})

		It("runs post-backup-unlock scripts on the deployment", func() {
			Expect(deployment.PostBackupUnlockCallCount()).To(Equal(1))
		})

		It("ensures that deployment is cleaned up", func() {
			Expect(deployment.CleanupCallCount()).To(Equal(1))
		})

		It("creates a local artifact", func() {
			Expect(artifactManager.CreateCallCount()).To(Equal(1))
		})

		It("names the artifact after the deployment", func() {
			actualDeploymentName, actualLogger := artifactManager.CreateArgsForCall(0)
			Expect(actualDeploymentName).To(Equal(deploymentName))
			Expect(actualLogger).To(Equal(logger))
		})

		It("drains the backup to the artifact", func() {
			Expect(deployment.CopyRemoteBackupToLocalCallCount()).To(Equal(1))
			Expect(deployment.CopyRemoteBackupToLocalArgsForCall(0)).To(Equal(artifact))
		})
	})

	Describe("failures", func() {
		var expectedError = fmt.Errorf("Profanity")
		var assertCleanupError = func() {
			var cleanupError = fmt.Errorf("gosh, it's a bit filthy in here")
			BeforeEach(func() {
				deployment.CleanupReturns(cleanupError)
			})

			It("includes the cleanup error in the returned error", func() {
				Expect(actualBackupError).To(MatchError(ContainSubstring(cleanupError.Error())))
			})
		}

		Context("when the artifact already exists", func() {
			BeforeEach(func() {
				artifactManager.ExistsReturns(true)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(
					ConsistOf(
						MatchError(fmt.Errorf("artifact %s already exists", deploymentName)),
					),
				)
			})
		})

		Context("fails to find deployment", func() {
			BeforeEach(func() {
				deploymentManager.FindReturns(nil, expectedError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(MatchError(expectedError)))
			})
		})

		Context("fails if manifest can't be saved", func() {
			var expectedError = fmt.Errorf("source of the nile")

			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)
				artifactManager.CreateReturns(artifact, nil)
				deploymentManager.SaveManifestReturns(expectedError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(expectedError))
			})

			It("cleans up", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("fails if the deployment is not backupable", func() {
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(false)
			})

			It("finds a deployment with the deployment name", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
				Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
			})

			It("checks if the deployment is backupable", func() {
				Expect(deployment.IsBackupableCallCount()).To(Equal(1))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(MatchError("Deployment '" + deploymentName + "' has no backup scripts")))
			})

			It("ensures that deployment is cleaned up", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})

			It("does not check the backup metadata validity", func() {
				Expect(deployment.HasValidBackupMetadataCallCount()).To(BeZero())
			})

			Context("cleanup fails as well", assertCleanupError)
		})

		Context("fails if pre-backup-lock fails", func() {
			var lockError = orchestrator.NewLockError("smoooooooth jazz")

			BeforeEach(func() {
				artifactManager.CreateReturns(artifact, nil)
				artifactManager.ExistsReturns(false)
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)
				deployment.CleanupReturns(nil)

				deployment.PreBackupLockReturns(lockError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(lockError))
			})

			It("also runs post-backup-unlock", func() {
				Expect(deployment.PostBackupUnlockCallCount()).To(Equal(1))
			})

			Context("cleanup fails as well", assertCleanupError)
		})

		Context("fails if post-backup-unlock fails", func() {
			var unlockError orchestrator.PostBackupUnlockError

			BeforeEach(func() {
				unlockError = orchestrator.NewPostBackupUnlockError("lalalalala")
				artifactManager.CreateReturns(artifact, nil)
				artifactManager.ExistsReturns(false)
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)
				deployment.CleanupReturns(nil)

				deployment.PostBackupUnlockReturns(unlockError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(unlockError))
			})

			It("continues with the cleanup", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})
			It("continues with drain artifact", func() {
				Expect(deployment.CopyRemoteBackupToLocalCallCount()).To(Equal(1))
			})

			Context("when the drain artifact fails as well", func() {
				var drainError = fmt.Errorf("just weird")

				BeforeEach(func() {
					deployment.CopyRemoteBackupToLocalReturns(drainError)
				})

				It("returns an error of type PostBackupUnlockError and "+
					"includes the drain error in the returned error", func() {
					Expect(actualBackupError).To(ConsistOf(drainError, unlockError))
				})

				Context("cleanup fails as well", func() {
					var cleanupError = orchestrator.NewCleanupError("here we go again")
					BeforeEach(func() {
						deployment.CleanupReturns(cleanupError)
					})

					It("includes the cleanup error in the returned error and "+
						"includes the drain error in the returned error and "+
						"includes the cleanup error in the returned error", func() {
						Expect(actualBackupError).To(ConsistOf(
							unlockError,
							drainError,
							And(
								BeAssignableToTypeOf(orchestrator.CleanupError{}),
								MatchError(ContainSubstring(cleanupError.Error())),
							),
						))
					})
				})
			})

			Context("cleanup fails as well", func() {
				var cleanupError = fmt.Errorf("leave me alone")
				BeforeEach(func() {
					deployment.CleanupReturns(cleanupError)
				})

				It("includes the cleanup error in the returned error "+
					"and returns an error of type PostBackupUnlockError", func() {
					Expect(actualBackupError).To(ConsistOf(
						MatchError(ContainSubstring(cleanupError.Error())),
						unlockError,
					))
				})
			})
		})

		Context("fails if backup cannot be drained", func() {
			var drainError = fmt.Errorf("I would like a sandwich")
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)
				artifactManager.CreateReturns(artifact, nil)
				deployment.CopyRemoteBackupToLocalReturns(drainError)
			})

			It("check if the deployment is backupable", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
				Expect(deployment.IsBackupableCallCount()).To(Equal(1))
			})

			It("backs up the deployment", func() {
				Expect(deployment.BackupCallCount()).To(Equal(1))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(drainError))
			})

			It("ensures that deployment's instance is cleaned up", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})

			Context("cleanup fails as well", assertCleanupError)
		})

		Context("fails if artifact cannot be created", func() {
			var artifactError = fmt.Errorf("I would like a sandwich")
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)

				artifactManager.CreateReturns(nil, artifactError)
			})

			It("should check if the deployment is backupable", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
			})

			It("dosent backup the deployment", func() {
				Expect(deployment.BackupCallCount()).To(BeZero())
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(artifactError))
			})

			It("ensures that deployment's instance is cleaned up", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})

			Context("cleanup fails as well", assertCleanupError)
		})

		Context("fails if the cleanup cannot be completed", func() {
			var cleanupError = fmt.Errorf("a tuna sandwich")
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)

				artifactManager.CreateReturns(artifact, nil)
				deployment.CleanupReturns(cleanupError)
			})

			It("should check if the deployment is backupable", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
			})

			It("backs up the deployment", func() {
				Expect(deployment.BackupCallCount()).To(Equal(1))
			})

			It("tries to cleanup the deployment instance", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(ContainSubstring(cleanupError.Error())))

			})
			It("returns a cleanup error", func() {
				Expect(actualBackupError).To(ConsistOf(BeAssignableToTypeOf(orchestrator.CleanupError{})))
			})

			Context("cleanup fails as well", assertCleanupError)
		})

		Context("fails if backup is not a success", func() {
			var backupError = fmt.Errorf("syzygy")
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)

				artifactManager.CreateReturns(artifact, nil)
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
				Expect(actualBackupError.Error()).To(ContainSubstring(backupError.Error()))
			})

			It("ensures that deployment's instance is cleaned up", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})

			Context("cleanup fails as well", assertCleanupError)
		})

		Context("fails if deployment is invalid", func() {
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(false)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(
					MatchError(fmt.Errorf("Multiple jobs in deployment '%s' specified the same backup name", deploymentName)),
				))
			})
		})
	})
})

var _ = Describe("CanBeBackedUp", func() {
	var (
		b                      *orchestrator.Backuper
		deployment             *fakes.FakeDeployment
		deploymentManager      *fakes.FakeDeploymentManager
		artifactManager        *fakes.FakeArtifactManager
		logger                 *fakes.FakeLogger
		deploymentName         = "foobarbaz"
		isDeploymentBackupable bool
	)

	BeforeEach(func() {
		deployment = new(fakes.FakeDeployment)
		deploymentManager = new(fakes.FakeDeploymentManager)
		artifactManager = new(fakes.FakeArtifactManager)
		logger = new(fakes.FakeLogger)
		b = orchestrator.NewBackuper(artifactManager, logger, deploymentManager)
	})

	JustBeforeEach(func() {
		isDeploymentBackupable, _ = b.CanBeBackedUp(deploymentName)
	})

	Context("when the deployment can be backed up", func() {
		BeforeEach(func() {
			artifactManager.ExistsReturns(false)
			deploymentManager.FindReturns(deployment, nil)
			deployment.IsBackupableReturns(true)
			deployment.HasValidBackupMetadataReturns(true)
			deployment.CleanupReturns(nil)
		})

		It("returns true", func() {
			Expect(isDeploymentBackupable).To(BeTrue())
		})

		It("finds the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the artifact already exists", func() {
			Expect(artifactManager.ExistsCallCount()).To(Equal(1))
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
			artifactManager.ExistsReturns(false)
			deploymentManager.FindReturns(nil, fmt.Errorf("deployment not found"))
			deployment.IsBackupableReturns(true)
			deployment.HasValidBackupMetadataReturns(true)
			deployment.CleanupReturns(nil)
		})

		It("returns false", func() {
			Expect(isDeploymentBackupable).To(BeFalse())
		})

		It("attempts to find the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})
	})
})
