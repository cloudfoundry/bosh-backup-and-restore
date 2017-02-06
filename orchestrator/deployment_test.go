package orchestrator_test

import (
	"fmt"
	"io"

	"bytes"

	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/pcf-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("Deployment", func() {
	var (
		deployment orchestrator.Deployment
		logger     orchestrator.Logger
		instances  []orchestrator.Instance
		instance1  *fakes.FakeInstance
		instance2  *fakes.FakeInstance
		instance3  *fakes.FakeInstance
	)

	JustBeforeEach(func() {
		deployment = orchestrator.NewBoshDeployment(logger, instances)
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
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)
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
				instance1.IsBackupableReturns(true)
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
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)
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
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(false)
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
				backupError := fmt.Errorf("My IQ is one of the highest — and you all know it!")
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)
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

	Context("IsBackupable", func() {
		var isBackupable bool

		JustBeforeEach(func() {
			isBackupable = deployment.IsBackupable()
		})

		Context("Single instance, backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})

			It("checks if the instance is backupable", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
			})

			It("returns true if the instance is backupable", func() {
				Expect(isBackupable).To(BeTrue())
			})
		})

		Context("Single instance, not backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})

			It("checks if the instance is backupable", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
			})

			It("returns true if any instance is backupable", func() {
				Expect(isBackupable).To(BeFalse())
			})
		})

		Context("Multiple instances, some backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance2.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true if any instance is backupable", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
				Expect(instance2.IsBackupableCallCount()).To(Equal(1))
				Expect(isBackupable).To(BeTrue())
			})
		})
		Context("Multiple instances, none backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance2.IsBackupableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true if any instance is backupable", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
				Expect(instance2.IsBackupableCallCount()).To(Equal(1))
				Expect(isBackupable).To(BeFalse())
			})
		})
	})

	Context("HasValidBackupMetadata", func() {
		var isValid bool

		JustBeforeEach(func() {
			isValid = deployment.HasValidBackupMetadata()
		})

		Context("Single instance, with unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBlobNamesReturns([]string{"custom1", "custom2"})
				instances = []orchestrator.Instance{instance1}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})

		Context("Single instance, with non-unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBlobNamesReturns([]string{"the-same", "the-same"})
				instances = []orchestrator.Instance{instance1}
			})

			It("returns false", func() {
				Expect(isValid).To(BeFalse())
			})
		})

		Context("multiple instances, with unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBlobNamesReturns([]string{"custom1", "custom2"})
				instance2.CustomBlobNamesReturns([]string{"custom3", "custom4"})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})

		Context("multiple instances, with non-unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBlobNamesReturns([]string{"custom1", "custom2"})
				instance2.CustomBlobNamesReturns([]string{"custom2", "custom4"})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns false", func() {
				Expect(isValid).To(BeFalse())
			})
		})

		Context("multiple instances, with no metadata", func() {
			BeforeEach(func() {
				instance1.CustomBlobNamesReturns([]string{})
				instance2.CustomBlobNamesReturns([]string{})
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
			var restoreError = fmt.Errorf("I have a plan, but I dont want to tell ISIS what it is")

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
			instance1.StreamBackupToRemoteReturns(nil)
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
				backupBlob.BackupChecksumReturns(blobCheckSum, nil)
				instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})
			})

			It("does not fail", func() {
				Expect(copyLocalBackupToRemoteError).NotTo(HaveOccurred())
			})

			It("checks the remote after transfer", func() {
				Expect(backupBlob.BackupChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(1))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupBlob))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupBlob.StreamBackupToRemoteCallCount()).To(Equal(1))
				expectedReader := backupBlob.StreamBackupToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupBlob.StreamBackupToRemoteReturns(fmt.Errorf("Tiny children are not horses"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("Tiny children are not horses"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("because i am smart")
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
					backupBlob.BackupChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					backupBlob.BackupChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadFileReturns(nil, fmt.Errorf("an overrated clown"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("an overrated clown"))
				})
			})
		})

		Context("multiple instances, one restorable", func() {
			var blobCheckSum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				instance2.IsRestorableReturns(false)

				artifact.FetchChecksumReturns(blobCheckSum, nil)
				backupBlob.BackupChecksumReturns(blobCheckSum, nil)
				instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})
			})

			It("does not fail", func() {
				Expect(copyLocalBackupToRemoteError).NotTo(HaveOccurred())
			})

			It("does not ask blobs for the non restorable instance", func() {
				Expect(instance2.BlobsCallCount()).To(BeZero())
			})

			It("checks the remote after transfer", func() {
				Expect(backupBlob.BackupChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(1))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupBlob))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupBlob.StreamBackupToRemoteCallCount()).To(Equal(1))
				expectedReader := backupBlob.StreamBackupToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupBlob.StreamBackupToRemoteReturns(fmt.Errorf("Tiny children are not horses"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("Tiny children are not horses"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("because i am smart")
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
					backupBlob.BackupChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					backupBlob.BackupChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadFileReturns(nil, fmt.Errorf("an overrated clown"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("an overrated clown"))
				})
			})
		})

		Context("Single instance, restorable, with multiple blobs", func() {
			var blobCheckSum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var anotherBackupBlob *fakes.FakeBackupBlob

			BeforeEach(func() {
				anotherBackupBlob = new(fakes.FakeBackupBlob)
				artifact.FetchChecksumReturns(blobCheckSum, nil)
				backupBlob.BackupChecksumReturns(blobCheckSum, nil)
				anotherBackupBlob.BackupChecksumReturns(blobCheckSum, nil)

				instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob, anotherBackupBlob})
			})

			It("does not fail", func() {
				Expect(copyLocalBackupToRemoteError).NotTo(HaveOccurred())
			})

			It("checks the remote after transfer", func() {
				Expect(backupBlob.BackupChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(2))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupBlob))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(anotherBackupBlob))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupBlob.StreamBackupToRemoteCallCount()).To(Equal(1))
				expectedReader := backupBlob.StreamBackupToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupBlob.StreamBackupToRemoteReturns(fmt.Errorf("Tiny children are not horses"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("Tiny children are not horses"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("because i am smart")
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
					backupBlob.BackupChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					backupBlob.BackupChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadFileReturns(nil, fmt.Errorf("an overrated clown"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("an overrated clown"))
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

				instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})
				instance1.IsBackupableReturns(true)
				artifact.CalculateChecksumReturns(remoteArtifactChecksum, nil)
				backupBlob.BackupChecksumReturns(remoteArtifactChecksum, nil)

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
				Expect(backupBlob.BackupChecksumCallCount()).To(Equal(1))
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

				instance1.BlobsReturns([]orchestrator.BackupBlob{blob1})
				instance2.BlobsReturns([]orchestrator.BackupBlob{blob2})

				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)

				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				instances = []orchestrator.Instance{instance1, instance2}
				blob1.BackupChecksumReturns(instanceChecksum, nil)
				blob2.BackupChecksumReturns(instanceChecksum, nil)
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
				Expect(blob1.BackupChecksumCallCount()).To(Equal(1))
				Expect(blob2.BackupChecksumCallCount()).To(Equal(1))
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

				instance1.IsBackupableReturns(true)
				instance1.BlobsReturns([]orchestrator.BackupBlob{remoteArtifact1})

				instance2.IsBackupableReturns(false)
				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				instances = []orchestrator.Instance{instance1, instance2}
				remoteArtifact1.BackupChecksumReturns(instanceChecksum, nil)
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
				Expect(remoteArtifact1.BackupChecksumCallCount()).To(Equal(1))
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
				var drainError = fmt.Errorf("they are bringing crime")

				BeforeEach(func() {
					backupBlob = new(fakes.FakeBackupBlob)

					instances = []orchestrator.Instance{instance1}
					instance1.IsBackupableReturns(true)
					instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})

					backupBlob.StreamFromRemoteReturns(drainError)
				})

				It("fails the transfer process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(drainError))
				})
			})

			Context("fails if file cannot be created", func() {
				var fileError = fmt.Errorf("i have a very good brain")
				BeforeEach(func() {
					instances = []orchestrator.Instance{instance1}
					instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})
					instance1.IsBackupableReturns(true)
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
					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})

					artifact.CreateFileReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(nil, shasumError)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(shasumError))
				})
			})

			Context("fails if the remote shasum cant be calulated", func() {
				remoteShasumError := fmt.Errorf("i have created so many jobs")
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})
					backupBlob.BackupChecksumReturns(nil, remoteShasumError)

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

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})

					artifact.CreateFileReturns(writeCloser1, nil)

					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this won't match"}, nil)
					backupBlob.BackupChecksumReturns(orchestrator.BackupChecksum{"file": "this wont match"}, nil)
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

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})

					artifact.CreateFileReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					backupBlob.BackupChecksumReturns(orchestrator.BackupChecksum{"file": "this will match"}, nil)
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

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.BlobsReturns([]orchestrator.BackupBlob{backupBlob})

					artifact.CreateFileReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					artifact.CalculateChecksumReturns(instanceChecksum, nil)
					backupBlob.BackupChecksumReturns(instanceChecksum, nil)

					backupBlob.DeleteReturns(expectedError)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(expectedError))
				})
			})
		})
	})
})
