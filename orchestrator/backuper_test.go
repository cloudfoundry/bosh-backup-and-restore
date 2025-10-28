package orchestrator_test

import (
	"fmt"

	"time"

	"github.com/cloudfoundry/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup", func() {
	var (
		b                     *orchestrator.Backuper
		deployment            *fakes.FakeDeployment
		deploymentManager     *fakes.FakeDeploymentManager
		fakeBackup            *fakes.FakeBackup
		fakeBackupManager     *fakes.FakeBackupManager
		logger                *fakes.FakeLogger
		lockOrderer           *fakes.FakeLockOrderer
		deploymentName        = "foobarbaz"
		actualBackupError     error
		startTime, finishTime time.Time
		artifactCopier        *fakes.FakeArtifactCopier
		timeStamp             string
		unsafeLockFree        bool
		nowFunc               func() time.Time
	)

	BeforeEach(func() {
		deployment = new(fakes.FakeDeployment)
		deploymentManager = new(fakes.FakeDeploymentManager)
		fakeBackupManager = new(fakes.FakeBackupManager)
		fakeBackup = new(fakes.FakeBackup)
		logger = new(fakes.FakeLogger)
		unsafeLockFree = false

		startTime = time.Now()
		finishTime = startTime.Add(time.Hour)
		timeStamp = time.Now().UTC().Format("20060102T150405Z")

		nows := []time.Time{startTime, finishTime}
		nowFunc = func() time.Time {
			var now time.Time
			now, nows = nows[0], nows[1:]
			return now
		}

		artifactCopier = new(fakes.FakeArtifactCopier)

	})

	JustBeforeEach(func() {
		b = orchestrator.NewBackuper(fakeBackupManager, logger, deploymentManager, lockOrderer, executor.NewParallelExecutor(), nowFunc, artifactCopier, unsafeLockFree, timeStamp)
		actualBackupError = b.Backup(deploymentName, "")
	})

	Context("backs up a deployment", func() {
		BeforeEach(func() {
			fakeBackupManager.CreateReturns(fakeBackup, nil)
			deploymentManager.FindReturns(deployment, nil)
			deployment.IsBackupableReturns(true)
			deployment.CleanupReturns(nil)
			artifactCopier.DownloadBackupFromDeploymentReturns(nil)
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
			Expect(actualArtifact).To(Equal(fakeBackup))
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
			afterSuccessfulBackup, _, _ := deployment.PostBackupUnlockArgsForCall(0)
			Expect(afterSuccessfulBackup).To(BeTrue())
		})

		It("ensures that deployment is cleaned up", func() {
			Expect(deployment.CleanupCallCount()).To(Equal(1))
		})

		It("creates a local artifact", func() {
			Expect(fakeBackupManager.CreateCallCount()).To(Equal(1))
		})

		It("names the artifact after the deployment", func() {
			actualPath, directoryName, actualLogger := fakeBackupManager.CreateArgsForCall(0)
			Expect(actualPath).To(Equal(""))
			Expect(directoryName).To(Equal(fmt.Sprintf("%s_%s", deploymentName, timeStamp)))
			Expect(actualLogger).To(Equal(logger))
		})

		It("drains the backup to the artifact", func() {
			Expect(artifactCopier.DownloadBackupFromDeploymentCallCount()).To(Equal(1))

			downloadedBackup, downloadedFromDeployment := artifactCopier.DownloadBackupFromDeploymentArgsForCall(0)
			Expect(downloadedBackup).To(Equal(fakeBackup))
			Expect(downloadedFromDeployment).To(Equal(deployment))
		})

		It("saves start and finish timestamps in the metadata file", func() {
			Expect(fakeBackup.CreateMetadataFileWithStartTimeArgsForCall(0)).To(Equal(startTime))
			Expect(fakeBackup.AddFinishTimeArgsForCall(0)).To(Equal(finishTime))
		})
	})

	Context("backs up a deployment without locking it", func() {
		BeforeEach(func() {
			unsafeLockFree = true
			fakeBackupManager.CreateReturns(fakeBackup, nil)
			deploymentManager.FindReturns(deployment, nil)
			deployment.IsBackupableReturns(true)
			deployment.CleanupReturns(nil)
			artifactCopier.DownloadBackupFromDeploymentReturns(nil)
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
			Expect(actualArtifact).To(Equal(fakeBackup))
		})

		It("checks if the deployment is backupable", func() {
			Expect(deployment.IsBackupableCallCount()).To(Equal(1))
		})

		It("runs pre-backup-lock scripts on the deployment", func() {
			Expect(deployment.PreBackupLockCallCount()).To(Equal(0))
		})

		It("runs backup scripts on the deployment", func() {
			Expect(deployment.BackupCallCount()).To(Equal(1))
		})

		It("runs post-backup-unlock scripts on the deployment", func() {
			Expect(deployment.PostBackupUnlockCallCount()).To(Equal(0))
		})

		It("ensures that deployment is cleaned up", func() {
			Expect(deployment.CleanupCallCount()).To(Equal(1))
		})

		It("creates a local artifact", func() {
			Expect(fakeBackupManager.CreateCallCount()).To(Equal(1))
		})

		It("names the artifact after the deployment", func() {
			actualPath, directoryName, actualLogger := fakeBackupManager.CreateArgsForCall(0)
			Expect(actualPath).To(Equal(""))
			Expect(directoryName).To(Equal(fmt.Sprintf("%s_%s", deploymentName, timeStamp)))
			Expect(actualLogger).To(Equal(logger))
		})

		It("drains the backup to the artifact", func() {
			Expect(artifactCopier.DownloadBackupFromDeploymentCallCount()).To(Equal(1))

			downloadedBackup, downloadedFromDeployment := artifactCopier.DownloadBackupFromDeploymentArgsForCall(0)
			Expect(downloadedBackup).To(Equal(fakeBackup))
			Expect(downloadedFromDeployment).To(Equal(deployment))
		})

		It("saves start and finish timestamps in the metadata file", func() {
			Expect(fakeBackup.CreateMetadataFileWithStartTimeArgsForCall(0)).To(Equal(startTime))
			Expect(fakeBackup.AddFinishTimeArgsForCall(0)).To(Equal(finishTime))
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

		Context("fails to find deployment", func() {
			BeforeEach(func() {
				deploymentManager.FindReturns(nil, expectedError)
			})

			It("fails the backup process", func() {
				expectErrorMatch(actualBackupError, expectedError)
			})
		})

		Context("fails if manifest can't be saved", func() {
			var expectedError = fmt.Errorf("source of the nile")

			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				fakeBackupManager.CreateReturns(fakeBackup, nil)
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

			Context("cleanup fails as well", assertCleanupError)
		})

		Context("fails if the artifact directory already exists", func() {
			BeforeEach(func() {
				fakeBackupManager.CreateReturns(fakeBackup, nil)
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.CheckArtifactDirReturns(fmt.Errorf("not ready"))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(ConsistOf(And(
					MatchError("not ready"),
					BeAssignableToTypeOf(orchestrator.ArtifactDirError{}),
				)))
			})
		})

		Context("fails if pre-backup-lock fails", func() {
			var lockError = orchestrator.NewLockError("smoooooooth jazz")

			BeforeEach(func() {
				fakeBackupManager.CreateReturns(fakeBackup, nil)
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.CleanupReturns(nil)

				deployment.PreBackupLockReturns(lockError)
			})

			It("fails the backup process", func() {
				expectErrorMatch(actualBackupError, lockError)
			})

			It("also runs post-backup-unlock", func() {
				Expect(deployment.PostBackupUnlockCallCount()).To(Equal(1))
				afterSuccessfulBackup, _, _ := deployment.PostBackupUnlockArgsForCall(0)
				Expect(afterSuccessfulBackup).To(BeFalse())
			})

			Context("cleanup fails as well", assertCleanupError)
		})

		Context("fails if post-backup-unlock fails", func() {
			var unlockError orchestrator.UnlockError

			BeforeEach(func() {
				unlockError = orchestrator.NewPostUnlockError("lalalalala")
				fakeBackupManager.CreateReturns(fakeBackup, nil)
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				deployment.CleanupReturns(nil)

				deployment.PostBackupUnlockReturns(unlockError)
			})

			It("returns the post backup unlock error", func() {
				expectErrorMatch(actualBackupError, unlockError)
			})

			It("continues with the cleanup", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})

			It("continues with drain artifact", func() {
				Expect(artifactCopier.DownloadBackupFromDeploymentCallCount()).To(Equal(1))
			})

			Context("when the drain artifact fails as well", func() {
				var drainError = fmt.Errorf("just weird")

				BeforeEach(func() {
					artifactCopier.DownloadBackupFromDeploymentReturns(drainError)
				})

				It("returns an error of type UnlockError and "+
					"includes the drain error in the returned error", func() {
					expectErrorMatch(actualBackupError, drainError, unlockError)
				})

				Context("cleanup fails as well", func() {
					var cleanupError = orchestrator.NewCleanupError("here we go again")
					BeforeEach(func() {
						deployment.CleanupReturns(cleanupError)
					})

					It("includes the cleanup error in the returned error and "+
						"includes the drain error in the returned error and "+
						"includes the cleanup error in the returned error", func() {
						expectErrorMatch(actualBackupError, drainError, unlockError, cleanupError)
					})
				})
			})

			Context("cleanup fails as well", func() {
				var cleanupError = fmt.Errorf("leave me alone")
				BeforeEach(func() {
					deployment.CleanupReturns(cleanupError)
				})

				It("includes the cleanup error in the returned error "+
					"and returns an error of type UnlockError", func() {
					expectErrorMatch(actualBackupError, unlockError, cleanupError)
				})
			})
		})

		Context("fails if backup cannot be drained", func() {
			var drainError = fmt.Errorf("I would like a sandwich")
			BeforeEach(func() {
				deploymentManager.FindReturns(deployment, nil)
				deployment.IsBackupableReturns(true)
				fakeBackupManager.CreateReturns(fakeBackup, nil)
				artifactCopier.DownloadBackupFromDeploymentReturns(drainError)
			})

			It("check if the deployment is backupable", func() {
				Expect(deploymentManager.FindCallCount()).To(Equal(1))
				Expect(deployment.IsBackupableCallCount()).To(Equal(1))
			})

			It("backs up the deployment", func() {
				Expect(deployment.BackupCallCount()).To(Equal(1))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError.Error()).To(ContainSubstring(drainError.Error()))
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

				fakeBackupManager.CreateReturns(nil, artifactError)
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

				fakeBackupManager.CreateReturns(fakeBackup, nil)
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

				fakeBackupManager.CreateReturns(fakeBackup, nil)
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
				Expect(fakeBackup.CreateArtifactCallCount()).To(BeZero())
			})

			It("fails the backup process", func() {
				Expect(actualBackupError.Error()).To(ContainSubstring(backupError.Error()))
			})

			It("runs post-backup-unlock scripts on the deployment", func() {
				Expect(deployment.PostBackupUnlockCallCount()).To(Equal(1))
				afterSuccessfulBackup, _, _ := deployment.PostBackupUnlockArgsForCall(0)
				Expect(afterSuccessfulBackup).To(BeFalse())
			})

			It("ensures that deployment's instance is cleaned up", func() {
				Expect(deployment.CleanupCallCount()).To(Equal(1))
			})

			Context("cleanup fails as well", assertCleanupError)
		})
	})
})

func expectErrorMatch(actual error, expected ...error) {
	if actualErrors, isErrorList := actual.(orchestrator.Error); isErrorList {
		for _, err := range actualErrors {
			Expect(actual).To(MatchError(ContainSubstring(err.Error())))
		}
		Expect(len(actualErrors)).To(Equal(len(expected)))
	} else {
		Expect(actual).To(MatchError(expected))
	}
}
