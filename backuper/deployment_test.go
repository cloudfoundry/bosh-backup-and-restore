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
				instance1.IsBackupableReturns(true, nil)
				instance1.PreBackupLockReturns(nil)
				instances = []backuper.Instance{instance1}
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
					instance1.IsBackupableReturns(true, nil)
					instance1.PreBackupLockReturns(lockErr)
				})

				It("fails", func() {
					Expect(lockErr).To(HaveOccurred())
				})
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
				instance1.IsBackupableReturns(true, nil)
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
				instance1.IsBackupableReturns(true, nil)
				instance2.IsBackupableReturns(true, nil)
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
				instance1.IsBackupableReturns(true, nil)
				instance2.IsBackupableReturns(false, nil)
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
				instance1.IsBackupableReturns(true, nil)
				instance2.IsBackupableReturns(true, nil)
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
				instance1.IsBackupableReturns(true, nil)
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
				instance1.IsBackupableReturns(false, nil)
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
				instance1.IsBackupableReturns(false, nil)
				instance2.IsBackupableReturns(true, nil)
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
				instance1.IsBackupableReturns(false, nil)
				instance2.IsBackupableReturns(false, nil)
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

		Context("Multiple instances, one fails to check if backupable", func() {
			var actualError = fmt.Errorf("No one has a higher IQ than me")
			BeforeEach(func() {
				instance1.IsBackupableReturns(false, actualError)
				instance2.IsBackupableReturns(true, nil)
				instances = []backuper.Instance{instance1, instance2}
			})
			It("fails", func() {
				Expect(isBackupableError).To(MatchError(actualError))
			})

			It("stops checking when an error occours", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
				Expect(instance2.IsBackupableCallCount()).To(Equal(0))
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
			copyRemoteBackupsToLocalArtifactError error
		)
		BeforeEach(func() {
			artifact = new(fakes.FakeArtifact)
		})
		JustBeforeEach(func() {
			copyRemoteBackupsToLocalArtifactError = deployment.CopyRemoteBackupToLocal(artifact)
		})

		Context("One instance, backupable", func() {
			var writeCloser *fakes.FakeWriteCloser
			var instanceChecksum = backuper.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			BeforeEach(func() {
				writeCloser = new(fakes.FakeWriteCloser)
				artifact.CreateFileReturns(writeCloser, nil)

				instance1.IsBackupableReturns(true, nil)
				artifact.CalculateChecksumReturns(instanceChecksum, nil)
				instances = []backuper.Instance{instance1}
				instance1.BackupChecksumReturns(instanceChecksum, nil)
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(1))
				Expect(artifact.CreateFileArgsForCall(0)).To(Equal(instance1))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(instance1.StreamBackupFromRemoteCallCount()).To(Equal(1))
				Expect(instance1.StreamBackupFromRemoteArgsForCall(0)).To(Equal(writeCloser))
			})

			It("closes the writer after its been streamed", func() {
				Expect(writeCloser.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the instance on the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(1))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(instance1))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(instance1.BackupChecksumCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(1))
				actualInstance, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualInstance).To(Equal(instance1))
				Expect(acutalChecksum).To(Equal(instanceChecksum))
			})
		})

		Context("Many instances, backupable", func() {
			var instanceChecksum = backuper.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writeCloser1 *fakes.FakeWriteCloser
			var writeCloser2 *fakes.FakeWriteCloser

			BeforeEach(func() {
				writeCloser1 = new(fakes.FakeWriteCloser)
				writeCloser2 = new(fakes.FakeWriteCloser)

				artifact.CreateFileStub = func(i backuper.InstanceIdentifer) (io.WriteCloser, error) {
					if i == instance1 {
						return writeCloser1, nil
					} else {
						return writeCloser2, nil
					}
				}

				instance1.IsBackupableReturns(true, nil)
				instance2.IsBackupableReturns(true, nil)
				artifact.CalculateChecksumReturns(instanceChecksum, nil)
				instances = []backuper.Instance{instance1, instance2}
				instance1.BackupChecksumReturns(instanceChecksum, nil)
				instance2.BackupChecksumReturns(instanceChecksum, nil)
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(2))
				Expect(artifact.CreateFileArgsForCall(0)).To(Equal(instance1))
				Expect(artifact.CreateFileArgsForCall(1)).To(Equal(instance2))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(instance1.StreamBackupFromRemoteCallCount()).To(Equal(1))
				Expect(instance1.StreamBackupFromRemoteArgsForCall(0)).To(Equal(writeCloser1))

				Expect(instance2.StreamBackupFromRemoteCallCount()).To(Equal(1))
				Expect(instance2.StreamBackupFromRemoteArgsForCall(0)).To(Equal(writeCloser2))
			})

			It("closes the writer after its been streamed", func() {
				Expect(writeCloser1.CloseCallCount()).To(Equal(1))
				Expect(writeCloser2.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the instance on the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(2))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(instance1))
				Expect(artifact.CalculateChecksumArgsForCall(1)).To(Equal(instance2))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(instance1.BackupChecksumCallCount()).To(Equal(1))
				Expect(instance2.BackupChecksumCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(2))
				actualInstance, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualInstance).To(Equal(instance1))
				Expect(acutalChecksum).To(Equal(instanceChecksum))

				actualInstance, acutalChecksum = artifact.AddChecksumArgsForCall(1)
				Expect(actualInstance).To(Equal(instance2))
				Expect(acutalChecksum).To(Equal(instanceChecksum))
			})
		})

		Context("Many instances, one backupable", func() {
			var instanceChecksum = backuper.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writeCloser1 *fakes.FakeWriteCloser

			BeforeEach(func() {
				writeCloser1 = new(fakes.FakeWriteCloser)

				artifact.CreateFileReturns(writeCloser1, nil)

				instance1.IsBackupableReturns(true, nil)
				instance2.IsBackupableReturns(false, nil)
				artifact.CalculateChecksumReturns(instanceChecksum, nil)
				instances = []backuper.Instance{instance1, instance2}
				instance1.BackupChecksumReturns(instanceChecksum, nil)
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(1))
				Expect(artifact.CreateFileArgsForCall(0)).To(Equal(instance1))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(instance1.StreamBackupFromRemoteCallCount()).To(Equal(1))
				Expect(instance1.StreamBackupFromRemoteArgsForCall(0)).To(Equal(writeCloser1))

				Expect(instance2.StreamBackupFromRemoteCallCount()).To(Equal(0))
			})

			It("closes the writer after its been streamed", func() {
				Expect(writeCloser1.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the instance on the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(1))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(instance1))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(instance1.BackupChecksumCallCount()).To(Equal(1))
				Expect(instance2.BackupChecksumCallCount()).To(Equal(0))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(1))
				actualInstance, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualInstance).To(Equal(instance1))
				Expect(acutalChecksum).To(Equal(instanceChecksum))
			})
		})

		Describe("failures", func() {
			var expectedError = fmt.Errorf("Jesus!")

			Context("fails when checking if instances are backupable", func() {
				BeforeEach(func() {
					instance1.IsBackupableReturns(false, expectedError)
					instances = []backuper.Instance{instance1}
				})

				It("fails the copy process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(expectedError))
				})
			})

			Context("fails if backup cannot be drained", func() {
				var drainError = fmt.Errorf("they are bringing crime")
				BeforeEach(func() {
					instances = []backuper.Instance{instance1}
					instance1.IsBackupableReturns(true, nil)
					instance1.StreamBackupFromRemoteReturns(drainError)
				})

				It("fails the transfer process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(drainError))
				})
			})

			Context("fails if file cannot be created", func() {
				var fileError = fmt.Errorf("i have a very good brain")
				BeforeEach(func() {
					instances = []backuper.Instance{instance1}
					instance1.IsBackupableReturns(true, nil)
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
					instance1.IsBackupableReturns(true, nil)
					instance1.BackupReturns(nil)
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

					instance1.IsBackupableReturns(true, nil)
					instance1.BackupReturns(nil)
					artifact.CreateFileReturns(writeCloser1, nil)

					instance1.BackupChecksumReturns(nil, remoteShasumError)
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

					instance1.IsBackupableReturns(true, nil)
					instance1.BackupReturns(nil)
					artifact.CreateFileReturns(writeCloser1, nil)

					artifact.CalculateChecksumReturns(backuper.BackupChecksum{"file": "this won't match"}, nil)
					instance1.BackupChecksumReturns(backuper.BackupChecksum{"file": "this wont match"}, nil)
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

					instance1.IsBackupableReturns(true, nil)
					instance1.BackupReturns(nil)
					artifact.CreateFileReturns(writeCloser1, nil)

					artifact.CalculateChecksumReturns(backuper.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					instance1.BackupChecksumReturns(backuper.BackupChecksum{"file": "this will match"}, nil)
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
