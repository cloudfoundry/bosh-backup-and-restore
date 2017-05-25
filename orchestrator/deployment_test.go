package orchestrator_test

import (
	"fmt"
	"io"

	"bytes"

	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("Deployment", func() {
	var (
		deployment orchestrator.Deployment
		logger     *fakes.FakeLogger
		instances  []orchestrator.Instance
		instance1  *fakes.FakeInstance
		instance2  *fakes.FakeInstance
		instance3  *fakes.FakeInstance
	)

	JustBeforeEach(func() {
		deployment = orchestrator.NewDeployment(logger, instances)
	})
	BeforeEach(func() {
		logger = new(fakes.FakeLogger)
		instance1 = new(fakes.FakeInstance)
		instance2 = new(fakes.FakeInstance)
		instance3 = new(fakes.FakeInstance)
	})

	Context("PreBackupLock", func() {
		var lockError error

		JustBeforeEach(func() {
			lockError = deployment.PreBackupLock()
		})

		Context("Single instance, backupable", func() {
			BeforeEach(func() {
				instance1.IsPreBackupLockableReturns(true)
				instance1.PreBackupLockReturns(nil)
				instances = []orchestrator.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(lockError).NotTo(HaveOccurred())
			})

			It("locks the instance", func() {
				Expect(instance1.PreBackupLockCallCount()).To(Equal(1))
			})

			Context("if the pre-backup-lock fails", func() {
				lockErr := fmt.Errorf("something")

				BeforeEach(func() {
					instance1.PreBackupLockReturns(lockErr)
				})

				It("fails", func() {
					Expect(lockErr).To(HaveOccurred())
				})
			})
		})

		Context("Multiple instances, some pre-backup-lockable", func() {
			BeforeEach(func() {
				instance1.HasBackupScriptReturns(true)
				instance2.HasBackupScriptReturns(true)
				instance1.IsPreBackupLockableReturns(true)
				instance2.IsPreBackupLockableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("does not fail", func() {
				Expect(lockError).NotTo(HaveOccurred())
			})

			It("runs pre-backup-lock on only the instance with the pre-backup-lock script", func() {
				Expect(instance1.PreBackupLockCallCount()).To(Equal(1))
				Expect(instance2.PreBackupLockCallCount()).To(Equal(0))
			})
		})

	})

	Context("Backup", func() {
		var (
			backupError error
		)
		JustBeforeEach(func() {
			backupError = deployment.Backup()
		})

		Context("Single instance, backupable", func() {
			BeforeEach(func() {
				instance1.HasBackupScriptReturns(true)
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(backupError).NotTo(HaveOccurred())
			})
			It("backs up the instance", func() {
				Expect(instance1.BackupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, all backupable", func() {
			BeforeEach(func() {
				instance1.HasBackupScriptReturns(true)
				instance2.HasBackupScriptReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(backupError).NotTo(HaveOccurred())
			})
			It("backs up the only the backupable instance", func() {
				Expect(instance1.BackupCallCount()).To(Equal(1))
				Expect(instance2.BackupCallCount()).To(Equal(1))
			})
		})
		Context("Multiple instances, some backupable", func() {
			BeforeEach(func() {
				instance1.HasBackupScriptReturns(true)
				instance2.HasBackupScriptReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(backupError).NotTo(HaveOccurred())
			})
			It("backs up the only the backupable instance", func() {
				Expect(instance1.BackupCallCount()).To(Equal(1))
			})
			It("does not back up the non backupable instance", func() {
				Expect(instance2.BackupCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, some failing to backup", func() {
			BeforeEach(func() {
				backupError := fmt.Errorf("very clever sandwich")
				instance1.HasBackupScriptReturns(true)
				instance2.HasBackupScriptReturns(true)
				instance1.BackupReturns(backupError)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(backupError).To(HaveOccurred())
			})

			It("stops invoking backup, after the error", func() {
				Expect(instance1.BackupCallCount()).To(Equal(1))
				Expect(instance2.BackupCallCount()).To(Equal(0))
			})
		})
	})

	Context("PostBackupUnlock", func() {
		var unlockError, expectedError error

		BeforeEach(func() {
			expectedError = fmt.Errorf("something went terribly wrong")
		})

		JustBeforeEach(func() {
			unlockError = deployment.PostBackupUnlock()
		})

		Context("Single instance, with post backup unlock", func() {
			BeforeEach(func() {
				instance1.IsPostBackupUnlockableReturns(true)
				instance1.PostBackupUnlockReturns(nil)
				instances = []orchestrator.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(unlockError).NotTo(HaveOccurred())
			})

			It("unlocks the instance", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
			})
		})

		Context("single instance, without post backup unlock", func() {
			BeforeEach(func() {
				instance1.IsPostBackupUnlockableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(unlockError).NotTo(HaveOccurred())
			})

			It("doesn't attempt to unlock the instance", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(0))
			})
		})

		Context("single instance that fails to unlock", func() {
			BeforeEach(func() {
				instance1.IsPostBackupUnlockableReturns(true)
				instance1.PostBackupUnlockReturns(expectedError)
				instances = []orchestrator.Instance{instance1}
			})

			It("fails", func() {
				Expect(unlockError).To(HaveOccurred())
			})

			It("attempts to unlock the instance", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, all with post backup unlock scripts", func() {
			BeforeEach(func() {
				instance1.IsPostBackupUnlockableReturns(true)
				instance1.PostBackupUnlockReturns(nil)
				instance2.IsPostBackupUnlockableReturns(true)
				instance2.PostBackupUnlockReturns(nil)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("does not fail", func() {
				Expect(unlockError).NotTo(HaveOccurred())
			})

			It("unlocks the instances", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
				Expect(instance2.PostBackupUnlockCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, one with post backup unlock scripts", func() {
			BeforeEach(func() {
				instance1.IsPostBackupUnlockableReturns(false)
				instance2.IsPostBackupUnlockableReturns(true)
				instance2.PostBackupUnlockReturns(nil)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("does not fail", func() {
				Expect(unlockError).NotTo(HaveOccurred())
			})

			It("unlocks the correct instance", func() {
				Expect(instance2.PostBackupUnlockCallCount()).To(Equal(1))
			})

			It("doesn't unlock the instance with no script", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, where one fails to unlock", func() {
			BeforeEach(func() {
				instance1.IsPostBackupUnlockableReturns(true)
				instance1.PostBackupUnlockReturns(expectedError)
				instance2.IsPostBackupUnlockableReturns(true)
				instance2.PostBackupUnlockReturns(nil)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				Expect(unlockError).To(HaveOccurred())
			})

			It("attempts to unlock both instances", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
				Expect(instance2.PostBackupUnlockCallCount()).To(Equal(1))
			})

			It("returns the expected single error", func() {
				Expect(unlockError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})

		Context("Multiple instances, all fail to unlock", func() {
			var secondError error

			BeforeEach(func() {
				instance1.IsPostBackupUnlockableReturns(true)
				instance1.PostBackupUnlockReturns(expectedError)

				secondError = fmt.Errorf("something else went wrong")
				instance2.IsPostBackupUnlockableReturns(true)
				instance2.PostBackupUnlockReturns(secondError)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				Expect(unlockError).To(HaveOccurred())
			})

			It("attempts to unlock both instances", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
				Expect(instance2.PostBackupUnlockCallCount()).To(Equal(1))
			})

			It("returns all the expected errors", func() {
				Expect(unlockError).To(MatchError(ContainSubstring(expectedError.Error())))
				Expect(unlockError).To(MatchError(ContainSubstring(secondError.Error())))
			})
		})
	})

	Context("HasBackupScript", func() {
		var hasBackupScript bool

		JustBeforeEach(func() {
			hasBackupScript = deployment.HasBackupScript()
		})

		Context("Single instance with a backup script", func() {
			BeforeEach(func() {
				instance1.HasBackupScriptReturns(true)
				instances = []orchestrator.Instance{instance1}
			})

			It("checks if the instance has a backup script", func() {
				Expect(instance1.HasBackupScriptCallCount()).To(Equal(1))
			})

			It("returns true", func() {
				Expect(hasBackupScript).To(BeTrue())
			})
		})

		Context("Single instance, no backup script", func() {
			BeforeEach(func() {
				instance1.HasBackupScriptReturns(false)
				instances = []orchestrator.Instance{instance1}
			})

			It("checks if the instance has a backup script", func() {
				Expect(instance1.HasBackupScriptCallCount()).To(Equal(1))
			})

			It("returns true", func() {
				Expect(hasBackupScript).To(BeFalse())
			})
		})

		Context("Multiple instances, some with backup scripts", func() {
			BeforeEach(func() {
				instance1.HasBackupScriptReturns(false)
				instance2.HasBackupScriptReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(instance1.HasBackupScriptCallCount()).To(Equal(1))
				Expect(instance2.HasBackupScriptCallCount()).To(Equal(1))
				Expect(hasBackupScript).To(BeTrue())
			})
		})
		Context("Multiple instances, none with backup scripts", func() {
			BeforeEach(func() {
				instance1.HasBackupScriptReturns(false)
				instance2.HasBackupScriptReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns false", func() {
				Expect(instance1.HasBackupScriptCallCount()).To(Equal(1))
				Expect(instance2.HasBackupScriptCallCount()).To(Equal(1))
				Expect(hasBackupScript).To(BeFalse())
			})
		})
	})

	Context("CheckArtifactDir", func() {
		var artifactDirError error

		BeforeEach(func() {
			instance1.NameReturns("foo")
			instance1.IDReturns("0")

			instance2.NameReturns("bar")
			instance2.IDReturns("0")

			instance1.ArtifactDirExistsReturns(false)
			instance2.ArtifactDirExistsReturns(false)
			instances = []orchestrator.Instance{instance1, instance2}
		})

		JustBeforeEach(func() {
			artifactDirError = deployment.CheckArtifactDir()
		})

		Context("when artifact directory does not exist", func() {
			It("does not fail", func() {
				Expect(artifactDirError).NotTo(HaveOccurred())
			})
		})

		Context("when artifact directory exists", func() {
			BeforeEach(func() {
				instance1.ArtifactDirExistsReturns(true)
				instance2.ArtifactDirExistsReturns(true)
			})

			It("fails", func() {
				Expect(artifactDirError).To(HaveOccurred())
			})

			It("the error includes the names of the instances on which the directory exists", func() {
				Expect(artifactDirError.Error()).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance foo/0"))
				Expect(artifactDirError.Error()).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance bar/0"))
			})
		})

	})

	Context("CustomArtifactNamesMatch", func() {
		var artifactMatchError error

		JustBeforeEach(func() {
			artifactMatchError = deployment.CustomArtifactNamesMatch()
		})
		BeforeEach(func() {
			instances = []orchestrator.Instance{instance1, instance2}
		})

		Context("when the custom names match", func() {
			BeforeEach(func() {
				instance1.CustomBackupBlobNamesReturns([]string{"custom1"})
				instance2.CustomRestoreBlobNamesReturns([]string{"custom1"})
			})

			It("is nil", func() {
				Expect(artifactMatchError).NotTo(HaveOccurred())
			})
		})

		Context("when the multiple custom names match", func() {
			BeforeEach(func() {
				instance1.CustomBackupBlobNamesReturns([]string{"custom1"})
				instance1.CustomRestoreBlobNamesReturns([]string{"custom2"})
				instance2.CustomBackupBlobNamesReturns([]string{"custom2"})
				instance2.CustomRestoreBlobNamesReturns([]string{"custom1"})
			})

			It("is nil", func() {
				Expect(artifactMatchError).NotTo(HaveOccurred())
			})
		})
		Context("when the custom dont match", func() {
			BeforeEach(func() {
				instance1.CustomBackupBlobNamesReturns([]string{"custom1"})
				instance2.NameReturns("job2Name")
				instance2.CustomRestoreBlobNamesReturns([]string{"custom2"})
			})

			It("to return an error", func() {
				Expect(artifactMatchError).To(MatchError("The job2Name restore script expects a backup script which produces custom2 artifact which is not present in the deployment."))
			})
		})
	})

	Context("HasUniqueCustomBackupNames", func() {
		var isValid bool

		JustBeforeEach(func() {
			isValid = deployment.HasUniqueCustomBackupNames()
		})

		Context("Single instance, with unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupBlobNamesReturns([]string{"custom1", "custom2"})
				instances = []orchestrator.Instance{instance1}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})

		Context("Single instance, with non-unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupBlobNamesReturns([]string{"the-same", "the-same"})
				instances = []orchestrator.Instance{instance1}
			})

			It("returns false", func() {
				Expect(isValid).To(BeFalse())
			})
		})

		Context("multiple instances, with unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupBlobNamesReturns([]string{"custom1", "custom2"})
				instance2.CustomBackupBlobNamesReturns([]string{"custom3", "custom4"})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})

		Context("multiple instances, with non-unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupBlobNamesReturns([]string{"custom1", "custom2"})
				instance2.CustomBackupBlobNamesReturns([]string{"custom2", "custom4"})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns false", func() {
				Expect(isValid).To(BeFalse())
			})
		})

		Context("multiple instances, with no metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupBlobNamesReturns([]string{})
				instance2.CustomBackupBlobNamesReturns([]string{})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})
	})

	Context("Restore", func() {
		var (
			restoreError error
		)
		JustBeforeEach(func() {
			restoreError = deployment.Restore()
		})

		Context("Single instance, restoreable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(restoreError).NotTo(HaveOccurred())
			})
			It("restores the instance", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(1))
			})
			It("logs before restoring", func() {
				_, message, _ := logger.InfoArgsForCall(0)
				Expect(message).To(Equal("Running restore scripts..."))
			})
		})
		Context("Single instance, not restoreable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(restoreError).NotTo(HaveOccurred())
			})
			It("restores the instance", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, all restoreable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instance2.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(restoreError).NotTo(HaveOccurred())
			})
			It("backs up the only the restoreable instance", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(1))
				Expect(instance2.RestoreCallCount()).To(Equal(1))
			})
		})
		Context("Multiple instances, some restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instance2.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(restoreError).NotTo(HaveOccurred())
			})
			It("backs up the only the restorable instance", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(1))
			})
			It("does not back up the non restorable instance", func() {
				Expect(instance2.RestoreCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, some failing to restore", func() {
			var restoreError = fmt.Errorf("and some salt and vinegar crisps")

			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instance2.IsRestorableReturns(true)
				instance1.RestoreReturns(restoreError)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(restoreError).To(MatchError(restoreError))
			})

			It("stops invoking backup, after the error", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(1))
				Expect(instance2.RestoreCallCount()).To(Equal(0))
			})
		})
	})

	Context("CopyLocalBackupToRemote", func() {
		var (
			artifact                     *fakes.FakeArtifact
			backupBlob                   *fakes.FakeBackupBlob
			copyLocalBackupToRemoteError error
			reader                       io.ReadCloser
		)

		BeforeEach(func() {
			reader = ioutil.NopCloser(bytes.NewBufferString("this-is-some-backup-data"))
			instance1.IsRestorableReturns(true)
			instances = []orchestrator.Instance{instance1}
			artifact = new(fakes.FakeArtifact)
			artifact.ReadFileReturns(reader, nil)
			backupBlob = new(fakes.FakeBackupBlob)
		})

		JustBeforeEach(func() {
			copyLocalBackupToRemoteError = deployment.CopyLocalBackupToRemote(artifact)
		})

		Context("Single instance, restorable", func() {
			var blobCheckSum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				artifact.FetchChecksumReturns(blobCheckSum, nil)
				backupBlob.ChecksumReturns(blobCheckSum, nil)
				instance1.BlobsToRestoreReturns([]orchestrator.BackupBlob{backupBlob})
			})

			It("does not fail", func() {
				Expect(copyLocalBackupToRemoteError).NotTo(HaveOccurred())
			})

			It("checks the remote after transfer", func() {
				Expect(backupBlob.ChecksumCallCount()).To(Equal(1))
			})

			It("marks the artifact directory as created after transfer", func() {
				Expect(instance1.MarkArtifactDirCreatedCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(1))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupBlob))
			})

			It("logs when the copy starts and finishes", func() {
				Expect(logger.InfoCallCount()).To(Equal(2))
				_, logLine, _ := logger.InfoArgsForCall(0)
				Expect(logLine).To(ContainSubstring("Copying backup"))

				_, logLine, _ = logger.InfoArgsForCall(1)
				Expect(logLine).To(ContainSubstring("Done"))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupBlob.StreamToRemoteCallCount()).To(Equal(1))
				expectedReader := backupBlob.StreamToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupBlob.StreamToRemoteReturns(fmt.Errorf("streaming had a problem"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("streaming had a problem"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("I am so clever")
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})
			Context("problem calculating shasum on remote", func() {
				var checksumError = fmt.Errorf("grr")
				BeforeEach(func() {
					backupBlob.ChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					backupBlob.ChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadFileReturns(nil, fmt.Errorf("leave me alone"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("leave me alone"))
				})
			})
		})

		Context("multiple instances, one restorable", func() {
			var blobCheckSum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				instance2.IsRestorableReturns(false)

				artifact.FetchChecksumReturns(blobCheckSum, nil)
				backupBlob.ChecksumReturns(blobCheckSum, nil)
				instance1.BlobsToRestoreReturns([]orchestrator.BackupBlob{backupBlob})
			})

			It("does not fail", func() {
				Expect(copyLocalBackupToRemoteError).NotTo(HaveOccurred())
			})

			It("does not ask blobs for the non restorable instance", func() {
				Expect(instance2.BlobsToRestoreCallCount()).To(BeZero())
			})

			It("checks the remote after transfer", func() {
				Expect(backupBlob.ChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(1))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupBlob))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupBlob.StreamToRemoteCallCount()).To(Equal(1))
				expectedReader := backupBlob.StreamToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupBlob.StreamToRemoteReturns(fmt.Errorf("I'm still here"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("I'm still here"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("oh well")
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})
			Context("problem calculating shasum on remote", func() {
				var checksumError = fmt.Errorf("grr")
				BeforeEach(func() {
					backupBlob.ChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					backupBlob.ChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadFileReturns(nil, fmt.Errorf("foo bar baz read error"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("foo bar baz read error"))
				})
			})
		})

		Context("Single instance, restorable, with multiple blobs", func() {
			var blobCheckSum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var anotherBackupBlob *fakes.FakeBackupBlob

			BeforeEach(func() {
				anotherBackupBlob = new(fakes.FakeBackupBlob)
				artifact.FetchChecksumReturns(blobCheckSum, nil)
				backupBlob.ChecksumReturns(blobCheckSum, nil)
				anotherBackupBlob.ChecksumReturns(blobCheckSum, nil)

				instance1.BlobsToRestoreReturns([]orchestrator.BackupBlob{backupBlob, anotherBackupBlob})
			})

			It("does not fail", func() {
				Expect(copyLocalBackupToRemoteError).NotTo(HaveOccurred())
			})

			It("checks the remote after transfer", func() {
				Expect(backupBlob.ChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(2))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupBlob))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(anotherBackupBlob))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupBlob.StreamToRemoteCallCount()).To(Equal(1))
				expectedReader := backupBlob.StreamToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupBlob.StreamToRemoteReturns(fmt.Errorf("this is a problem"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("this is a problem"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("checksum error occurred")
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})
			Context("problem calculating shasum on remote", func() {
				var checksumError = fmt.Errorf("grr")
				BeforeEach(func() {
					backupBlob.ChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					backupBlob.ChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadFileReturns(nil, fmt.Errorf("a huge problem"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("a huge problem"))
				})
			})
		})

	})

	Context("IsRestorable", func() {
		var (
			isRestorableError error
			isRestorable      bool
		)
		JustBeforeEach(func() {
			isRestorable = deployment.IsRestorable()
		})

		Context("Single instance, restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
			})

			It("checks if the instance is restorable", func() {
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
			})

			It("returns true if the instance is restorable", func() {
				Expect(isRestorable).To(BeTrue())
			})
		})
		Context("Single instance, not restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
			})

			It("checks if the instance is restorable", func() {
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
			})

			It("returns true if any instance is restorable", func() {
				Expect(isRestorable).To(BeFalse())
			})
		})

		Context("Multiple instances, some restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instance2.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
			})

			It("returns true if any instance is restorable", func() {
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
				Expect(instance2.IsRestorableCallCount()).To(Equal(1))
				Expect(isRestorable).To(BeTrue())
			})
		})
		Context("Multiple instances, none restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instance2.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
			})

			It("returns true if any instance is restorable", func() {
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
				Expect(instance2.IsRestorableCallCount()).To(Equal(1))
				Expect(isRestorable).To(BeFalse())
			})
		})
	})
	Context("Cleanup", func() {
		var (
			actualCleanupError error
		)
		JustBeforeEach(func() {
			actualCleanupError = deployment.Cleanup()
		})

		Context("Single instance", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("cleans up the instance", func() {
				Expect(instance1.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("backs up the only the backupable instance", func() {
				Expect(instance1.CleanupCallCount()).To(Equal(1))
				Expect(instance2.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, some failing", func() {
			var cleanupError1 = fmt.Errorf("foo")

			BeforeEach(func() {
				instance1.CleanupReturns(cleanupError1)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
			})

			It("continues cleanup of instances", func() {
				Expect(instance1.CleanupCallCount()).To(Equal(1))
				Expect(instance2.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, all failing", func() {
			var cleanupError1 = fmt.Errorf("foo")
			var cleanupError2 = fmt.Errorf("bar")
			BeforeEach(func() {
				instance1.CleanupReturns(cleanupError1)
				instance2.CleanupReturns(cleanupError2)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails with both error messages", func() {
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError2.Error())))
			})

			It("continues cleanup of instances", func() {
				Expect(instance1.CleanupCallCount()).To(Equal(1))
				Expect(instance2.CleanupCallCount()).To(Equal(1))
			})
		})
	})
	Context("Instances", func() {
		BeforeEach(func() {
			instances = []orchestrator.Instance{instance1, instance2, instance3}
		})
		It("returns instances for the deployment", func() {
			Expect(deployment.Instances()).To(ConsistOf(instance1, instance2, instance3))
		})
	})

	Context("CopyRemoteBackupsToLocal", func() {
		var (
			artifact                              *fakes.FakeArtifact
			backupBlob                            *fakes.FakeBackupBlob
			copyRemoteBackupsToLocalArtifactError error
		)
		BeforeEach(func() {
			artifact = new(fakes.FakeArtifact)
			backupBlob = new(fakes.FakeBackupBlob)
		})
		JustBeforeEach(func() {
			copyRemoteBackupsToLocalArtifactError = deployment.CopyRemoteBackupToLocal(artifact)
		})

		Context("One instance, backupable", func() {
			var localArtifactWriteCloser *fakes.FakeWriteCloser
			var remoteArtifactChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				localArtifactWriteCloser = new(fakes.FakeWriteCloser)
				artifact.CreateFileReturns(localArtifactWriteCloser, nil)

				instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{backupBlob})
				instance1.HasBackupScriptReturns(true)
				artifact.CalculateChecksumReturns(remoteArtifactChecksum, nil)
				backupBlob.ChecksumReturns(remoteArtifactChecksum, nil)

				instances = []orchestrator.Instance{instance1}
			})

			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(1))
				Expect(artifact.CreateFileArgsForCall(0)).To(Equal(backupBlob))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(backupBlob.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(backupBlob.StreamFromRemoteArgsForCall(0)).To(Equal(localArtifactWriteCloser))
			})

			It("closes the writer after its been streamed", func() {
				Expect(localArtifactWriteCloser.CloseCallCount()).To(Equal(1))
			})

			It("deletes the blob on the remote", func() {
				Expect(backupBlob.DeleteCallCount()).To(Equal(1))
			})

			It("calculates checksum for the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(1))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(backupBlob))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(backupBlob.ChecksumCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(1))
				actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(backupBlob))
				Expect(acutalChecksum).To(Equal(remoteArtifactChecksum))
			})
		})

		Context("Many instances, backupable", func() {
			var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writeCloser1 *fakes.FakeWriteCloser
			var writeCloser2 *fakes.FakeWriteCloser

			var blob1 *fakes.FakeBackupBlob
			var blob2 *fakes.FakeBackupBlob

			BeforeEach(func() {
				writeCloser1 = new(fakes.FakeWriteCloser)
				writeCloser2 = new(fakes.FakeWriteCloser)
				blob1 = new(fakes.FakeBackupBlob)
				blob2 = new(fakes.FakeBackupBlob)

				artifact.CreateFileStub = func(i orchestrator.BackupBlobIdentifier) (io.WriteCloser, error) {
					if i == blob1 {
						return writeCloser1, nil
					} else {
						return writeCloser2, nil
					}
				}

				instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{blob1})
				instance2.BlobsToBackupReturns([]orchestrator.BackupBlob{blob2})

				instance1.HasBackupScriptReturns(true)
				instance2.HasBackupScriptReturns(true)

				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				instances = []orchestrator.Instance{instance1, instance2}
				blob1.ChecksumReturns(instanceChecksum, nil)
				blob2.ChecksumReturns(instanceChecksum, nil)
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(2))
				Expect(artifact.CreateFileArgsForCall(0)).To(Equal(blob1))
				Expect(artifact.CreateFileArgsForCall(1)).To(Equal(blob2))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(blob1.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(blob1.StreamFromRemoteArgsForCall(0)).To(Equal(writeCloser1))

				Expect(blob2.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(blob2.StreamFromRemoteArgsForCall(0)).To(Equal(writeCloser2))
			})

			It("closes the writer after its been streamed", func() {
				Expect(writeCloser1.CloseCallCount()).To(Equal(1))
				Expect(writeCloser2.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the instance on the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(2))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(blob1))
				Expect(artifact.CalculateChecksumArgsForCall(1)).To(Equal(blob2))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(blob1.ChecksumCallCount()).To(Equal(1))
				Expect(blob2.ChecksumCallCount()).To(Equal(1))
			})

			It("deletes both the blobs", func() {
				Expect(blob1.DeleteCallCount()).To(Equal(1))
				Expect(blob2.DeleteCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(2))
				actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(blob1))
				Expect(acutalChecksum).To(Equal(instanceChecksum))

				actualRemoteArtifact, acutalChecksum = artifact.AddChecksumArgsForCall(1)
				Expect(actualRemoteArtifact).To(Equal(blob2))
				Expect(acutalChecksum).To(Equal(instanceChecksum))
			})
		})

		Context("Many instances, one backupable", func() {
			var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writeCloser1 *fakes.FakeWriteCloser
			var remoteArtifact1 *fakes.FakeBackupBlob

			BeforeEach(func() {
				writeCloser1 = new(fakes.FakeWriteCloser)
				remoteArtifact1 = new(fakes.FakeBackupBlob)

				artifact.CreateFileReturns(writeCloser1, nil)

				instance1.HasBackupScriptReturns(true)
				instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{remoteArtifact1})

				instance2.HasBackupScriptReturns(false)
				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				instances = []orchestrator.Instance{instance1, instance2}
				remoteArtifact1.ChecksumReturns(instanceChecksum, nil)
			})
			It("succeeds", func() {
				Expect(copyRemoteBackupsToLocalArtifactError).To(Succeed())
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(1))
				Expect(artifact.CreateFileArgsForCall(0)).To(Equal(remoteArtifact1))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(remoteArtifact1.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(remoteArtifact1.StreamFromRemoteArgsForCall(0)).To(Equal(writeCloser1))
			})

			It("closes the writer after its been streamed", func() {
				Expect(writeCloser1.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the instance on the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(1))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(remoteArtifact1))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(remoteArtifact1.ChecksumCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(1))
				actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(remoteArtifact1))
				Expect(acutalChecksum).To(Equal(instanceChecksum))
			})
		})

		Describe("failures", func() {
			Context("fails if backup cannot be drained", func() {
				var drainError = fmt.Errorf("please make it stop")

				BeforeEach(func() {
					backupBlob = new(fakes.FakeBackupBlob)

					instances = []orchestrator.Instance{instance1}
					instance1.HasBackupScriptReturns(true)
					instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{backupBlob})

					backupBlob.StreamFromRemoteReturns(drainError)
				})

				It("fails the transfer process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(drainError))
				})
			})

			Context("fails if file cannot be created", func() {
				var fileError = fmt.Errorf("not a good file")
				BeforeEach(func() {
					instances = []orchestrator.Instance{instance1}
					instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{backupBlob})
					instance1.HasBackupScriptReturns(true)
					artifact.CreateFileReturns(nil, fileError)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(fileError))
				})
			})

			Context("fails if local shasum calculation fails", func() {
				shasumError := fmt.Errorf("yuuuge")
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}
					instance1.HasBackupScriptReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{backupBlob})

					artifact.CreateFileReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(nil, shasumError)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(shasumError))
				})
			})

			Context("fails if the remote shasum cant be calulated", func() {
				remoteShasumError := fmt.Errorf("this shasum is not happy")
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}

					instance1.HasBackupScriptReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{backupBlob})
					backupBlob.ChecksumReturns(nil, remoteShasumError)

					artifact.CreateFileReturns(writeCloser1, nil)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(remoteShasumError))
				})

				It("dosen't try to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})

			Context("fails if the remote shasum dosen't match the local shasum", func() {
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}

					instance1.HasBackupScriptReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{backupBlob})

					artifact.CreateFileReturns(writeCloser1, nil)

					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this won't match"}, nil)
					backupBlob.ChecksumReturns(orchestrator.BackupChecksum{"file": "this wont match"}, nil)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(ContainSubstring("Backup artifact is corrupted")))
				})

				It("dosen't try to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})

			Context("fails if the number of files in the artifact dont match", func() {
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}

					instance1.HasBackupScriptReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{backupBlob})

					artifact.CreateFileReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					backupBlob.ChecksumReturns(orchestrator.BackupChecksum{"file": "this will match"}, nil)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(ContainSubstring("Backup artifact is corrupted")))
				})

				It("dosen't try to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})

			Context("fails if unable to delete blobs", func() {
				var writeCloser1 *fakes.FakeWriteCloser
				var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
				var expectedError = fmt.Errorf("brr")

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}

					instance1.HasBackupScriptReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsToBackupReturns([]orchestrator.BackupBlob{backupBlob})

					artifact.CreateFileReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					artifact.CalculateChecksumReturns(instanceChecksum, nil)
					backupBlob.ChecksumReturns(instanceChecksum, nil)

					backupBlob.DeleteReturns(expectedError)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(expectedError))
				})
			})
		})
	})
})
