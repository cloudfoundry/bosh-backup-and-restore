package orchestrator_test

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
)

var _ = Describe("ArtifactCopier", func() {
	var (
		artifactCopier orchestrator.ArtifactCopier
		logger         *fakes.FakeLogger
		deployment     *fakes.FakeDeployment
		localBackup    *fakes.FakeBackup
		err            error

		instance1 *fakes.FakeInstance
		instance2 *fakes.FakeInstance

		job1a *fakes.FakeJob
		job1b *fakes.FakeJob
		job2a *fakes.FakeJob
		job3a *fakes.FakeJob

		remoteBackup1 *fakes.FakeBackupArtifact
		remoteBackup2 *fakes.FakeBackupArtifact

		instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
	)

	BeforeEach(func() {
		logger = new(fakes.FakeLogger)

		job1a = new(fakes.FakeJob)
		job1b = new(fakes.FakeJob)
		job2a = new(fakes.FakeJob)
		job3a = new(fakes.FakeJob)

		instance1 = new(fakes.FakeInstance)
		instance2 = new(fakes.FakeInstance)
		instance1.JobsReturns([]orchestrator.Job{job1a, job1b})
		instance2.JobsReturns([]orchestrator.Job{job2a})

		deployment = new(fakes.FakeDeployment)
		deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1, instance2})

		localBackup = new(fakes.FakeBackup)
		localBackup.CalculateChecksumReturns(instanceChecksum, nil)
		localBackup.FetchChecksumReturns(instanceChecksum, nil)

		remoteBackup1 = new(fakes.FakeBackupArtifact)
		remoteBackup2 = new(fakes.FakeBackupArtifact)

		remoteBackup1.NameReturns("remote_backup_artifact_1")
		remoteBackup1.InstanceNameReturns("instance1")
		remoteBackup1.InstanceIDReturns("0")
		remoteBackup1.ChecksumReturns(instanceChecksum, nil)
		remoteBackup2.NameReturns("remote_backup_artifact_2")
		remoteBackup2.InstanceNameReturns("instance2")
		remoteBackup2.InstanceIDReturns("0")
		remoteBackup2.ChecksumReturns(instanceChecksum, nil)

		artifactCopier = orchestrator.NewArtifactCopier(executor.NewSerialExecutor(), logger)
	})

	Context("DownloadBackupFromDeployment", func() {
		var (
			writer1 *fakes.FakeWriteCloser
			writer2 *fakes.FakeWriteCloser
		)

		BeforeEach(func() {
			writer1 = new(fakes.FakeWriteCloser)
			writer2 = new(fakes.FakeWriteCloser)

			localBackup.CreateArtifactStub = func(i orchestrator.ArtifactIdentifier) (io.WriteCloser, error) {
				if i == remoteBackup1 {
					return writer1, nil
				} else {
					return writer2, nil
				}
			}

			instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{remoteBackup1})
			instance2.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{remoteBackup2})
		})

		JustBeforeEach(func() {
			err = artifactCopier.DownloadBackupFromDeployment(localBackup, deployment)
		})

		It("downloads artifacts from all backupable instances", func() {
			By("creating an artifact file with the instance", func() {
				Expect(localBackup.CreateArtifactCallCount()).To(Equal(2))
				Expect(localBackup.CreateArtifactArgsForCall(0)).To(Equal(remoteBackup1))
				Expect(localBackup.CreateArtifactArgsForCall(1)).To(Equal(remoteBackup2))
			})

			By("streaming the backup to the writer for the artifact file", func() {
				Expect(remoteBackup1.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(remoteBackup1.StreamFromRemoteArgsForCall(0)).To(Equal(writer1))

				Expect(remoteBackup2.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(remoteBackup2.StreamFromRemoteArgsForCall(0)).To(Equal(writer2))
			})

			By("closing the writer after its been streamed", func() {
				Expect(writer1.CloseCallCount()).To(Equal(1))
				Expect(writer2.CloseCallCount()).To(Equal(1))
			})

			By("calculating checksum for the instance on the artifact", func() {
				Expect(localBackup.CalculateChecksumCallCount()).To(Equal(2))
				Expect(localBackup.CalculateChecksumArgsForCall(0)).To(Equal(remoteBackup1))
				Expect(localBackup.CalculateChecksumArgsForCall(1)).To(Equal(remoteBackup2))
			})

			By("calculating checksum for the instance on remote", func() {
				Expect(remoteBackup1.ChecksumCallCount()).To(Equal(1))
				Expect(remoteBackup2.ChecksumCallCount()).To(Equal(1))
			})

			By("deleting both the artifacts", func() {
				Expect(remoteBackup1.DeleteCallCount()).To(Equal(1))
				Expect(remoteBackup2.DeleteCallCount()).To(Equal(1))
			})

			By("appending the checksum for the instance on the artifact", func() {
				Expect(localBackup.AddChecksumCallCount()).To(Equal(2))
				actualRemoteArtifact, actualChecksum := localBackup.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(remoteBackup1))
				Expect(actualChecksum).To(Equal(instanceChecksum))

				actualRemoteArtifact, actualChecksum = localBackup.AddChecksumArgsForCall(1)
				Expect(actualRemoteArtifact).To(Equal(remoteBackup2))
				Expect(actualChecksum).To(Equal(instanceChecksum))
			})
		})

		Context("when a backup cannot be drained", func() {
			var drainError = fmt.Errorf("please make it stop")

			BeforeEach(func() {
				remoteBackup2.StreamFromRemoteReturns(drainError)
			})

			It("fails the transfer process", func() {
				Expect(err).To(MatchError(ContainSubstring("please make it stop")))
			})
		})

		Context("when a file cannot be created", func() {
			var fileError = fmt.Errorf("not a good file")

			BeforeEach(func() {
				localBackup.CreateArtifactReturns(nil, fileError)
			})

			It("fails the backup process", func() {
				Expect(err).To(MatchError(ContainSubstring("not a good file")))
			})
		})

		Context("when a local shasum calculation fails", func() {
			shasumError := fmt.Errorf("yuuuge")

			BeforeEach(func() {
				localBackup.CalculateChecksumReturns(nil, shasumError)
			})

			It("fails the backup process", func() {
				Expect(err).To(MatchError(ContainSubstring("yuuuge")))
			})
		})

		Context("when a remote shasum can't be calculated", func() {
			remoteShasumError := fmt.Errorf("this shasum is not happy")

			BeforeEach(func() {
				remoteBackup1.ChecksumReturns(nil, remoteShasumError)
			})

			It("fails the backup process", func() {
				Expect(err).To(MatchError(ContainSubstring("this shasum is not happy")))
			})

			It("only tries to append the valid shasum to metadata", func() {
				Expect(localBackup.AddChecksumCallCount()).To(Equal(1))
				actualRemoteArtifact, actualChecksum := localBackup.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(remoteBackup2))
				Expect(actualChecksum).To(Equal(instanceChecksum))
			})
		})

		Context("when a remote shasum doesn't match the local shasum", func() {
			BeforeEach(func() {
				remoteBackup2.ChecksumReturns(orchestrator.BackupChecksum{"file1": "abcd", "file2": "DOES_NOT_MATCH"}, nil)
			})

			It("fails", func() {
				By("printing an error", func() {
					Expect(err).To(MatchError(ContainSubstring("Backup is corrupted")))
					Expect(err).To(MatchError(ContainSubstring("instance2/0 remote_backup_artifact_2 - checksums don't match for [file2]")))
					Expect(err).To(MatchError(ContainSubstring("Checksum failed for 1 files in total")))
					Expect(err).To(MatchError(Not(ContainSubstring("file1"))))
				})

				By("only appending the checksum for the instance when the checksum matches", func() {
					Expect(localBackup.AddChecksumCallCount()).To(Equal(1))
					actualRemoteArtifact, actualChecksum := localBackup.AddChecksumArgsForCall(0)
					Expect(actualRemoteArtifact).To(Equal(remoteBackup1))
					Expect(actualChecksum).To(Equal(instanceChecksum))
				})

				By("only deleting the artifact when the checksum matches", func() {
					Expect(remoteBackup1.DeleteCallCount()).To(Equal(1))
					Expect(remoteBackup2.DeleteCallCount()).To(Equal(0))
				})
			})
		})

		Context("when the number of files in the artifact doesn't match", func() {
			BeforeEach(func() {
				localBackup.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
				remoteBackup1.ChecksumReturns(orchestrator.BackupChecksum{"file": "this will match"}, nil)
			})

			It("fails the backup process", func() {
				Expect(err).To(MatchError(ContainSubstring("Backup is corrupted")))
			})

			It("doesn't try to append shasum to metadata", func() {
				Expect(localBackup.AddChecksumCallCount()).To(BeZero())
			})
		})

		Context("when unable to delete artifacts", func() {
			var expectedError = fmt.Errorf("unable to delete file error")

			BeforeEach(func() {
				remoteBackup1.DeleteReturns(expectedError)
			})

			It("fails the backup process", func() {
				Expect(err).To(MatchError(ContainSubstring("unable to delete file error")))
			})
		})
	})

	Context("UploadBackupToDeployment", func() {
		var (
			reader1 io.ReadCloser
			reader2 io.ReadCloser
		)

		BeforeEach(func() {
			reader1 = ioutil.NopCloser(bytes.NewBufferString("this-is-some-backup-data"))
			reader2 = ioutil.NopCloser(bytes.NewBufferString("this-is-some-other-backup-data"))
			deployment.RestorableInstancesReturns([]orchestrator.Instance{instance1})

			localBackup.ReadArtifactStub = func(i orchestrator.ArtifactIdentifier) (io.ReadCloser, error) {
				if i == remoteBackup1 {
					return reader1, nil
				} else {
					return reader2, nil
				}
			}

			instance1.ArtifactsToRestoreReturns([]orchestrator.BackupArtifact{remoteBackup1, remoteBackup2})
		})

		JustBeforeEach(func() {
			err = artifactCopier.UploadBackupToDeployment(localBackup, deployment)
		})

		It("uploads artifacts to all restorable instances", func() {
			By("not failing", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			By("checking the remote after transfer", func() {
				Expect(remoteBackup1.ChecksumCallCount()).To(Equal(1))
				Expect(remoteBackup2.ChecksumCallCount()).To(Equal(1))
			})

			By("checking the local checksum", func() {
				Expect(localBackup.FetchChecksumCallCount()).To(Equal(2))
				Expect(localBackup.FetchChecksumArgsForCall(0)).To(Equal(remoteBackup1))
				Expect(localBackup.FetchChecksumArgsForCall(1)).To(Equal(remoteBackup2))
			})

			By("streaming the backup file to the restorable instance", func() {
				Expect(remoteBackup1.StreamToRemoteCallCount()).To(Equal(1))
				Expect(remoteBackup1.StreamToRemoteArgsForCall(0)).To(Equal(reader1))
				Expect(remoteBackup2.StreamToRemoteCallCount()).To(Equal(1))
				Expect(remoteBackup2.StreamToRemoteArgsForCall(0)).To(Equal(reader2))
			})
		})

		Context("when a problem occurs while streaming to an instance", func() {
			BeforeEach(func() {
				remoteBackup1.StreamToRemoteReturns(fmt.Errorf("this is a problem"))
			})

			It("fails", func() {
				Expect(err).To(MatchError(ContainSubstring("this is a problem")))
			})
		})

		Context("when a problem occurs calculating shasum on local", func() {
			BeforeEach(func() {
				localBackup.FetchChecksumReturns(nil, fmt.Errorf("checksum error occurred"))
			})

			It("fails", func() {
				Expect(err).To(MatchError(ContainSubstring("checksum error occurred")))
			})
		})

		Context("a problem occurs calculating shasum on remote", func() {
			BeforeEach(func() {
				remoteBackup1.ChecksumReturns(nil, fmt.Errorf("grr"))
			})

			It("fails", func() {
				Expect(err).To(MatchError(ContainSubstring("grr")))
			})
		})

		Context("when shas don't match after transfer", func() {
			BeforeEach(func() {
				remoteBackup1.ChecksumReturns(orchestrator.BackupChecksum{"file1": "abcd", "file2": "thisdoesnotmatch"}, nil)
			})

			It("fails", func() {
				Expect(err).To(MatchError(ContainSubstring("Backup couldn't be transferred, checksum failed")))
			})

			It("lists only the files with mismatched checksums", func() {
				Expect(err).To(MatchError(ContainSubstring("file2")))
				Expect(err).NotTo(MatchError(ContainSubstring("file1")))
			})

			It("prints the number of files with mismatched checksums", func() {
				Expect(err).To(MatchError(ContainSubstring("Checksum failed for 1 files in total")))
			})
		})

		Context("when a problem occurs while reading from backup", func() {
			BeforeEach(func() {
				localBackup.ReadArtifactReturns(nil, fmt.Errorf("a huge problem"))
			})

			It("fails", func() {
				Expect(err).To(MatchError(ContainSubstring("a huge problem")))
			})
		})
	})
})
