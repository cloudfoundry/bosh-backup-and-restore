package backuper_test

import (
	"fmt"
	"io"

	"bytes"

	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"
)

var _ = Describe("Deployment", func() {
	var (
		deployment backuper.Deployment
		logger     backuper.Logger
		instances  []backuper.Instance
		instance1  *fakes.FakeInstance
		instance2  *fakes.FakeInstance
		instance3  *fakes.FakeInstance
	)

	JustBeforeEach(func() {
		deployment = backuper.NewBoshDeployment(logger, instances)
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
				instance1.IsPreBackupLockableReturns(true, nil)
				instance1.PreBackupLockReturns(nil)
				instances = []backuper.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(lockError).NotTo(HaveOccurred())
			})

			It("locks the instance", func() {
				Expect(instance1.PreBackupLockCallCount()).To(Equal(1))
			})

			Context("if checking for the pre-backup-lock scripts fails", func() {
				checkErr := fmt.Errorf("foobar")

				BeforeEach(func() {
					instance1.IsPreBackupLockableReturns(false, checkErr)
				})

				It("fails", func() {
					Expect(lockError).To(HaveOccurred())
					Expect(lockError).To(MatchError(ContainSubstring(checkErr.Error())))
				})
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
				instance1.IsPreBackupLockableReturns(true, nil)
				instance2.IsPreBackupLockableReturns(false, nil)
				instances = []backuper.Instance{instance1, instance2}
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
				instances = []backuper.Instance{instance1}
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
				instances = []backuper.Instance{instance1, instance2}
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
				instances = []backuper.Instance{instance1, instance2}
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
				instances = []backuper.Instance{instance1, instance2}
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
				instance1.IsPostBackupUnlockableReturns(true, nil)
				instance1.PostBackupUnlockReturns(nil)
				instances = []backuper.Instance{instance1}
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
				instance1.IsPostBackupUnlockableReturns(false, nil)
				instances = []backuper.Instance{instance1}
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
				instance1.IsPostBackupUnlockableReturns(true, nil)
				instance1.PostBackupUnlockReturns(expectedError)
				instances = []backuper.Instance{instance1}
			})

			It("fails", func() {
				Expect(unlockError).To(HaveOccurred())
			})

			It("attempts to unlock the instance", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
			})
		})

		Context("single instance that fails checking for post-backup-unlock scripts", func() {
			var checkUnlockableError = fmt.Errorf("i know a lot about hacking")

			BeforeEach(func() {
				instance1.IsPostBackupUnlockableReturns(false, checkUnlockableError)
				instances = []backuper.Instance{instance1}
			})

			It("fails", func() {
				Expect(unlockError).To(HaveOccurred())
				Expect(unlockError).To(MatchError(ContainSubstring(checkUnlockableError.Error())))
			})

			It("does not attempt to unlock the instance", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, all with post backup unlock scripts", func() {
			BeforeEach(func() {
				instance1.IsPostBackupUnlockableReturns(true, nil)
				instance1.PostBackupUnlockReturns(nil)
				instance2.IsPostBackupUnlockableReturns(true, nil)
				instance2.PostBackupUnlockReturns(nil)
				instances = []backuper.Instance{instance1, instance2}
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
				instance1.IsPostBackupUnlockableReturns(false, nil)
				instance2.IsPostBackupUnlockableReturns(true, nil)
				instance2.PostBackupUnlockReturns(nil)
				instances = []backuper.Instance{instance1, instance2}
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
				instance1.IsPostBackupUnlockableReturns(true, nil)
				instance1.PostBackupUnlockReturns(expectedError)
				instance2.IsPostBackupUnlockableReturns(true, nil)
				instance2.PostBackupUnlockReturns(nil)
				instances = []backuper.Instance{instance1, instance2}
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
				instance1.IsPostBackupUnlockableReturns(true, nil)
				instance1.PostBackupUnlockReturns(expectedError)

				secondError = fmt.Errorf("something else went wrong")
				instance2.IsPostBackupUnlockableReturns(true, nil)
				instance2.PostBackupUnlockReturns(secondError)
				instances = []backuper.Instance{instance1, instance2}
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
		var (
			isBackupableError error
			isBackupable      bool
		)
		JustBeforeEach(func() {
			isBackupable, isBackupableError = deployment.IsBackupable()
		})

		Context("Single instance, backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instances = []backuper.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(isBackupableError).NotTo(HaveOccurred())
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
				instances = []backuper.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(isBackupableError).NotTo(HaveOccurred())
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
				instances = []backuper.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(isBackupableError).NotTo(HaveOccurred())
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
				instances = []backuper.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(isBackupableError).NotTo(HaveOccurred())
			})

			It("returns true if any instance is backupable", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
				Expect(instance2.IsBackupableCallCount()).To(Equal(1))
				Expect(isBackupable).To(BeFalse())
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
				instance1.IsRestorableReturns(true, nil)
				instances = []backuper.Instance{instance1}
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
				instance1.IsRestorableReturns(false, nil)
				instances = []backuper.Instance{instance1}
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
				instance1.IsRestorableReturns(true, nil)
				instance2.IsRestorableReturns(true, nil)
				instances = []backuper.Instance{instance1, instance2}
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
				instance1.IsRestorableReturns(true, nil)
				instance2.IsRestorableReturns(false, nil)
				instances = []backuper.Instance{instance1, instance2}
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
				instance1.IsRestorableReturns(true, nil)
				instance2.IsRestorableReturns(true, nil)
				instance1.RestoreReturns(restoreError)
				instances = []backuper.Instance{instance1, instance2}
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
			artifact    *fakes.FakeArtifact
			loadFromErr error
			reader      io.ReadCloser
		)

		BeforeEach(func() {
			instance1.IsRestorableReturns(true, nil)
			instance1.StreamBackupToRemoteReturns(nil)
			instances = []backuper.Instance{instance1}
			artifact = new(fakes.FakeArtifact)
			artifact.ReadFileReturns(reader, nil)
		})

		JustBeforeEach(func() {
			reader = ioutil.NopCloser(bytes.NewBufferString("this-is-some-backup-data"))
			loadFromErr = deployment.CopyLocalBackupToRemote(artifact)
		})

		Context("Single instance, restorable", func() {
			var instanceChecksum = backuper.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				artifact.FetchChecksumReturns(instanceChecksum, nil)
				instance1.BackupChecksumReturns(instanceChecksum, nil)
			})

			It("does not fail", func() {
				Expect(loadFromErr).NotTo(HaveOccurred())
			})

			It("checks the remote after transfer", func() {
				Expect(instance1.BackupChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(1))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(instance1))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(instance1.StreamBackupToRemoteCallCount()).To(Equal(1))
				expectedReader := instance1.StreamBackupToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					instance1.StreamBackupToRemoteReturns(fmt.Errorf("Tiny children are not horses"))
				})

				It("fails", func() {
					Expect(loadFromErr).To(HaveOccurred())
					Expect(loadFromErr).To(MatchError("Tiny children are not horses"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("because i am smart")
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(loadFromErr).To(HaveOccurred())
					Expect(loadFromErr).To(MatchError(checksumError))
				})
			})
			Context("problem calculating shasum on remote", func() {
				var checksumError = fmt.Errorf("grr")
				BeforeEach(func() {
					instance1.BackupChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(loadFromErr).To(HaveOccurred())
					Expect(loadFromErr).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					instance1.BackupChecksumReturns(backuper.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(loadFromErr).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadFileReturns(nil, fmt.Errorf("an overrated clown"))
				})

				It("fails", func() {
					Expect(loadFromErr).To(HaveOccurred())
					Expect(loadFromErr).To(MatchError("an overrated clown"))
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
			isRestorable, isRestorableError = deployment.IsRestorable()
		})

		Context("Single instance, restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true, nil)
				instances = []backuper.Instance{instance1}
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
				instance1.IsRestorableReturns(false, nil)
				instances = []backuper.Instance{instance1}
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
				instance1.IsRestorableReturns(false, nil)
				instance2.IsRestorableReturns(true, nil)
				instances = []backuper.Instance{instance1, instance2}
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
				instance1.IsRestorableReturns(false, nil)
				instance2.IsRestorableReturns(false, nil)
				instances = []backuper.Instance{instance1, instance2}
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

		Context("Multiple instances, one fails to check if restorable", func() {
			var actualError = fmt.Errorf("No one has a higher IQ than me")
			BeforeEach(func() {
				instance1.IsRestorableReturns(false, actualError)
				instance2.IsRestorableReturns(true, nil)
				instances = []backuper.Instance{instance1, instance2}
			})
			It("fails", func() {
				Expect(isRestorableError).To(MatchError(actualError))
			})

			It("stops checking when an error occours", func() {
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
				Expect(instance2.IsRestorableCallCount()).To(Equal(0))
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
				instances = []backuper.Instance{instance1}
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
				instances = []backuper.Instance{instance1, instance2}
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
				instances = []backuper.Instance{instance1, instance2}
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
				instances = []backuper.Instance{instance1, instance2}
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
			instances = []backuper.Instance{instance1, instance2, instance3}
		})
		It("returns instances for the deployment", func() {
			Expect(deployment.Instances()).To(ConsistOf(instance1, instance2, instance3))
		})
	})

	Context("CopyRemoteBackupsToLocal", func() {
		var (
			artifact                              *fakes.FakeArtifact
			remoteArtifact *fakes.FakeRemoteArtifact
			copyRemoteBackupsToLocalArtifactError error
		)
		BeforeEach(func() {
			artifact = new(fakes.FakeArtifact)
			remoteArtifact = new(fakes.FakeRemoteArtifact)
		})
		JustBeforeEach(func() {
			copyRemoteBackupsToLocalArtifactError = deployment.CopyRemoteBackupToLocal(artifact)
		})

		Context("One instance, backupable", func() {
			var localArtifactWriteCloser *fakes.FakeWriteCloser
			var remoteArtifactChecksum = backuper.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				localArtifactWriteCloser = new(fakes.FakeWriteCloser)
				artifact.CreateFileReturns(localArtifactWriteCloser, nil)

				instance1.RemoteArtifactReturns(remoteArtifact)
				instance1.IsBackupableReturns(true)
				artifact.CalculateChecksumReturns(remoteArtifactChecksum, nil)
				remoteArtifact.BackupChecksumReturns(remoteArtifactChecksum, nil)

				instances = []backuper.Instance{instance1}
			})

			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(1))
				Expect(artifact.CreateFileArgsForCall(0)).To(Equal(remoteArtifact))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(remoteArtifact.StreamBackupFromRemoteCallCount()).To(Equal(1))
				Expect(remoteArtifact.StreamBackupFromRemoteArgsForCall(0)).To(Equal(localArtifactWriteCloser))
			})

			It("closes the writer after its been streamed", func() {
				Expect(localArtifactWriteCloser.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(1))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(remoteArtifact))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(remoteArtifact.BackupChecksumCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(1))
				actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(remoteArtifact))
				Expect(acutalChecksum).To(Equal(remoteArtifactChecksum))
			})
		})

		Context("Many instances, backupable", func() {
			var instanceChecksum = backuper.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writeCloser1 *fakes.FakeWriteCloser
			var writeCloser2 *fakes.FakeWriteCloser

			var remoteArtifact1 *fakes.FakeRemoteArtifact
			var remoteArtifact2 *fakes.FakeRemoteArtifact

			BeforeEach(func() {
				writeCloser1 = new(fakes.FakeWriteCloser)
				writeCloser2 = new(fakes.FakeWriteCloser)
				remoteArtifact1 = new(fakes.FakeRemoteArtifact)
				remoteArtifact2 = new(fakes.FakeRemoteArtifact)

				artifact.CreateFileStub = func(i backuper.InstanceIdentifer) (io.WriteCloser, error) {
					if i == remoteArtifact1 {
						return writeCloser1, nil
					} else {
						return writeCloser2, nil
					}
				}

				instance1.RemoteArtifactReturns(remoteArtifact1)
				instance2.RemoteArtifactReturns(remoteArtifact2)

				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)

				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				instances = []backuper.Instance{instance1, instance2}
				remoteArtifact1.BackupChecksumReturns(instanceChecksum, nil)
				remoteArtifact2.BackupChecksumReturns(instanceChecksum, nil)
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(2))
				Expect(artifact.CreateFileArgsForCall(0)).To(Equal(remoteArtifact1))
				Expect(artifact.CreateFileArgsForCall(1)).To(Equal(remoteArtifact2))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(remoteArtifact1.StreamBackupFromRemoteCallCount()).To(Equal(1))
				Expect(remoteArtifact1.StreamBackupFromRemoteArgsForCall(0)).To(Equal(writeCloser1))

				Expect(remoteArtifact2.StreamBackupFromRemoteCallCount()).To(Equal(1))
				Expect(remoteArtifact2.StreamBackupFromRemoteArgsForCall(0)).To(Equal(writeCloser2))
			})

			It("closes the writer after its been streamed", func() {
				Expect(writeCloser1.CloseCallCount()).To(Equal(1))
				Expect(writeCloser2.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the instance on the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(2))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(remoteArtifact1))
				Expect(artifact.CalculateChecksumArgsForCall(1)).To(Equal(remoteArtifact2))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(remoteArtifact1.BackupChecksumCallCount()).To(Equal(1))
				Expect(remoteArtifact2.BackupChecksumCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(2))
				actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(remoteArtifact1))
				Expect(acutalChecksum).To(Equal(instanceChecksum))

				actualRemoteArtifact, acutalChecksum = artifact.AddChecksumArgsForCall(1)
				Expect(actualRemoteArtifact).To(Equal(remoteArtifact2))
				Expect(acutalChecksum).To(Equal(instanceChecksum))
			})
		})

		Context("Many instances, one backupable", func() {
			var instanceChecksum = backuper.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writeCloser1 *fakes.FakeWriteCloser
			var remoteArtifact1 *fakes.FakeRemoteArtifact

			BeforeEach(func() {
				writeCloser1 = new(fakes.FakeWriteCloser)
				remoteArtifact1 = new(fakes.FakeRemoteArtifact)

				artifact.CreateFileReturns(writeCloser1, nil)

				instance1.IsBackupableReturns(true)
				instance1.RemoteArtifactReturns(remoteArtifact1)

				instance2.IsBackupableReturns(false)
				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				instances = []backuper.Instance{instance1, instance2}
				remoteArtifact1.BackupChecksumReturns(instanceChecksum, nil)
			})
			It("succeeds",func(){
				Expect(copyRemoteBackupsToLocalArtifactError).To(Succeed())
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(1))
				Expect(artifact.CreateFileArgsForCall(0)).To(Equal(remoteArtifact1))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(remoteArtifact1.StreamBackupFromRemoteCallCount()).To(Equal(1))
				Expect(remoteArtifact1.StreamBackupFromRemoteArgsForCall(0)).To(Equal(writeCloser1))
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
					remoteArtifact = new(fakes.FakeRemoteArtifact)

					instances = []backuper.Instance{instance1}
					instance1.IsBackupableReturns(true)
					instance1.RemoteArtifactReturns(remoteArtifact)

					remoteArtifact.StreamBackupFromRemoteReturns(drainError)
				})

				It("fails the transfer process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(drainError))
				})
			})

			Context("fails if file cannot be created", func() {
				var fileError = fmt.Errorf("i have a very good brain")
				BeforeEach(func() {
					instances = []backuper.Instance{instance1}
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
					instances = []backuper.Instance{instance1}
					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.RemoteArtifactReturns(remoteArtifact)

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
					instances = []backuper.Instance{instance1}

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.RemoteArtifactReturns(remoteArtifact)
					remoteArtifact.BackupChecksumReturns(nil, remoteShasumError)

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
					instances = []backuper.Instance{instance1}

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.RemoteArtifactReturns(remoteArtifact)

					artifact.CreateFileReturns(writeCloser1, nil)

					artifact.CalculateChecksumReturns(backuper.BackupChecksum{"file": "this won't match"}, nil)
					remoteArtifact.BackupChecksumReturns(backuper.BackupChecksum{"file": "this wont match"}, nil)
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
					instances = []backuper.Instance{instance1}

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.RemoteArtifactReturns(remoteArtifact)

					artifact.CreateFileReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(backuper.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					remoteArtifact.BackupChecksumReturns(backuper.BackupChecksum{"file": "this will match"}, nil)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(ContainSubstring("Backup artifact is corrupted")))
				})

				It("dosen't try to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})
		})
	})
})
