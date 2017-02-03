package orchestrator_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/pcf-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("Backuper", func() {
	var (
		boshDirector       *fakes.FakeBoshDirector
		b                  *orchestrator.Backuper
		deployment         *fakes.FakeDeployment
		deploymentManager  *fakes.FakeDeploymentManager
		artifact           *fakes.FakeArtifact
		artifactManager    *fakes.FakeArtifactManager
		logger             *fakes.FakeLogger
		deploymentName     = "foobarbaz"
		deploymentManifest = "what a magnificent manifest"
		actualBackupError  orchestrator.Error
	)

	BeforeEach(func() {
		deployment = new(fakes.FakeDeployment)
		deploymentManager = new(fakes.FakeDeploymentManager)
		boshDirector = new(fakes.FakeBoshDirector)
		artifactManager = new(fakes.FakeArtifactManager)
		artifact = new(fakes.FakeArtifact)
		logger = new(fakes.FakeLogger)
		b = orchestrator.NewBackuper(boshDirector, artifactManager, logger, deploymentManager)
	})

	JustBeforeEach(func() {
		actualBackupError = b.Backup(deploymentName)
	})

	Context("backs up a deployment", func() {
		BeforeEach(func() {
			boshDirector.GetManifestReturns(deploymentManifest, nil)
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
			Expect(boshDirector.GetManifestCallCount()).To(Equal(1))
			Expect(boshDirector.GetManifestArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the artifact already exists", func() {
			Expect(artifactManager.ExistsCallCount()).To(Equal(1))
		})

		It("checks if the deployment is backupable", func() {
			Expect(deployment.IsBackupableCallCount()).To(Equal(1))
		})

		It("runs p-pre-backup-lock scripts on the deployment", func() {
			Expect(deployment.PreBackupLockCallCount()).To(Equal(1))
		})

		It("runs p-backup scripts on the deployment", func() {
			Expect(deployment.BackupCallCount()).To(Equal(1))
		})

		It("runs p-post-backup-unlock scripts on the deployment", func() {
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
		var expectedError = fmt.Errorf("Jesus!")
		var assertCleanupError = func() {
			var cleanupError = fmt.Errorf("he was born in kenya")
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

		Context("fails if manifest can't be downloaded", func() {
			var expectedError = fmt.Errorf("he the founder of isis")
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)
				artifactManager.CreateReturns(artifact, nil)
				boshDirector.GetManifestReturns("", expectedError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(expectedError))
			})
		})

		Context("fails if manifest can't be saved", func() {
			var expectedError = fmt.Errorf("he the founder of isis")

			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)
				artifactManager.CreateReturns(artifact, nil)
				boshDirector.GetManifestReturns(deploymentManifest, nil)
				artifact.SaveManifestReturns(expectedError)
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
			var lockError = fmt.Errorf("it was going to be a smooth transition - NOT")

			BeforeEach(func() {
				boshDirector.GetManifestReturns(deploymentManifest, nil)
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
			var unlockError error

			BeforeEach(func() {
				unlockError = fmt.Errorf("it was going to be a smooth transition - NOT")
				boshDirector.GetManifestReturns(deploymentManifest, nil)
				artifactManager.CreateReturns(artifact, nil)
				artifactManager.ExistsReturns(false)
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.HasValidBackupMetadataReturns(true)
				deployment.CleanupReturns(nil)

				deployment.PostBackupUnlockReturns(unlockError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(ContainSubstring(unlockError.Error())))
			})

			It("fails with the correct error type", func() {
				Expect(actualBackupError).To(ConsistOf(BeAssignableToTypeOf(orchestrator.PostBackupUnlockError{})))
			})

			It("continues with the cleanup", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})
			It("continues with drain artifact", func() {
				Expect(deployment.CopyRemoteBackupToLocalCallCount()).To(Equal(1))
			})

			Context("when the drain artifact fails as well", func() {
				var drainError = fmt.Errorf("i don't do email but i know about hacking")

				BeforeEach(func() {
					deployment.CopyRemoteBackupToLocalReturns(drainError)
				})

				It("returns an error of type PostBackupUnlockError and "+
					"includes the drain error in the returned error", func() {
					Expect(actualBackupError).To(ConsistOf(drainError, BeAssignableToTypeOf(orchestrator.PostBackupUnlockError{})))
				})

				Context("cleanup fails as well", func() {
					var cleanupError = fmt.Errorf("he was born in kenya")
					BeforeEach(func() {
						deployment.CleanupReturns(cleanupError)
					})

					It("includes the cleanup error in the returned error and "+
						"includes the drain error in the returned error and "+
						"includes the cleanup error in the returned error", func() {
						Expect(actualBackupError).To(ConsistOf(
							BeAssignableToTypeOf(orchestrator.PostBackupUnlockError{}),
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
				var cleanupError = fmt.Errorf("he was born in kenya")
				BeforeEach(func() {
					deployment.CleanupReturns(cleanupError)
				})

				It("includes the cleanup error in the returned error "+
					"and returns an error of type PostBackupUnlockError", func() {
					Expect(actualBackupError).To(ConsistOf(
						MatchError(ContainSubstring(cleanupError.Error())),
						BeAssignableToTypeOf(orchestrator.PostBackupUnlockError{}),
					))
				})
			})
		})

		Context("fails if backup cannot be drained", func() {
			var drainError = fmt.Errorf("they are bringing crime")
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
			var artifactError = fmt.Errorf("they are bringing crime")
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
			var cleanupError = fmt.Errorf("why doesn't he show his birth certificate?")
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
			var backupError = fmt.Errorf("i have the best words")
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
				Expect(actualBackupError).To(ConsistOf(backupError))
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
