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

		instance1 *fakes.FakeInstance
		instance2 *fakes.FakeInstance
		instance3 *fakes.FakeInstance

		job1a *fakes.FakeJob
		job1b *fakes.FakeJob
		job2a *fakes.FakeJob
		job3a *fakes.FakeJob
	)

	BeforeEach(func() {
		logger = new(fakes.FakeLogger)

		instance1 = new(fakes.FakeInstance)
		instance2 = new(fakes.FakeInstance)
		instance3 = new(fakes.FakeInstance)

		job1a = new(fakes.FakeJob)
		job1b = new(fakes.FakeJob)
		job2a = new(fakes.FakeJob)
		job3a = new(fakes.FakeJob)

		instance1.JobsReturns([]orchestrator.Job{job1a, job1b})
		instance2.JobsReturns([]orchestrator.Job{job2a})
		instance3.JobsReturns([]orchestrator.Job{job3a})

		deployment = new(fakes.FakeDeployment)

		artifactCopier = orchestrator.NewArtifactCopier(executor.NewSerialExecutor(), logger)
	})

	Context("DownloadBackupFromDeployment", func() {
		var (
			artifact       *fakes.FakeBackup
			backupArtifact *fakes.FakeBackupArtifact
			err            error
		)

		BeforeEach(func() {
			artifact = new(fakes.FakeBackup)
			backupArtifact = new(fakes.FakeBackupArtifact)
		})

		JustBeforeEach(func() {
			err = artifactCopier.DownloadBackupFromDeployment(artifact, deployment)
		})

		Context("One instance, backupable", func() {
			var localArtifactWriteCloser *fakes.FakeWriteCloser
			var remoteArtifactChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				localArtifactWriteCloser = new(fakes.FakeWriteCloser)
				artifact.CreateArtifactReturns(localArtifactWriteCloser, nil)

				instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})
				artifact.CalculateChecksumReturns(remoteArtifactChecksum, nil)

				backupArtifact.ChecksumReturns(remoteArtifactChecksum, nil)

				deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1})
			})

			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateArtifactCallCount()).To(Equal(1))
				Expect(artifact.CreateArtifactArgsForCall(0)).To(Equal(backupArtifact))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(backupArtifact.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(backupArtifact.StreamFromRemoteArgsForCall(0)).To(Equal(localArtifactWriteCloser))
			})

			It("closes the writer after its been streamed", func() {
				Expect(localArtifactWriteCloser.CloseCallCount()).To(Equal(1))
			})

			It("deletes the artifact on the remote", func() {
				Expect(backupArtifact.DeleteCallCount()).To(Equal(1))
			})

			It("calculates checksum for the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(1))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(backupArtifact))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(backupArtifact.ChecksumCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(1))
				actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(backupArtifact))
				Expect(acutalChecksum).To(Equal(remoteArtifactChecksum))
			})
		})

		Context("Many instances, backupable", func() {
			var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writer1 *fakes.FakeWriteCloser
			var writer2 *fakes.FakeWriteCloser

			var remoteArtifact1 *fakes.FakeBackupArtifact
			var remoteArtifact2 *fakes.FakeBackupArtifact

			BeforeEach(func() {
				writer1 = new(fakes.FakeWriteCloser)
				writer2 = new(fakes.FakeWriteCloser)
				remoteArtifact1 = new(fakes.FakeBackupArtifact)
				remoteArtifact2 = new(fakes.FakeBackupArtifact)

				artifact.CreateArtifactStub = func(i orchestrator.ArtifactIdentifier) (io.WriteCloser, error) {
					if i == remoteArtifact1 {
						return writer1, nil
					} else {
						return writer2, nil
					}
				}

				instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{remoteArtifact1})
				instance2.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{remoteArtifact2})

				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1, instance2})
				remoteArtifact1.ChecksumReturns(instanceChecksum, nil)
				remoteArtifact2.ChecksumReturns(instanceChecksum, nil)
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateArtifactCallCount()).To(Equal(2))
				Expect(artifact.CreateArtifactArgsForCall(0)).To(Equal(remoteArtifact1))
				Expect(artifact.CreateArtifactArgsForCall(1)).To(Equal(remoteArtifact2))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(remoteArtifact1.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(remoteArtifact1.StreamFromRemoteArgsForCall(0)).To(Equal(writer1))

				Expect(remoteArtifact2.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(remoteArtifact2.StreamFromRemoteArgsForCall(0)).To(Equal(writer2))
			})

			It("closes the writer after its been streamed", func() {
				Expect(writer1.CloseCallCount()).To(Equal(1))
				Expect(writer2.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the instance on the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(2))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(remoteArtifact1))
				Expect(artifact.CalculateChecksumArgsForCall(1)).To(Equal(remoteArtifact2))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(remoteArtifact1.ChecksumCallCount()).To(Equal(1))
				Expect(remoteArtifact2.ChecksumCallCount()).To(Equal(1))
			})

			It("deletes both the artifacts", func() {
				Expect(remoteArtifact1.DeleteCallCount()).To(Equal(1))
				Expect(remoteArtifact2.DeleteCallCount()).To(Equal(1))
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
			var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writeCloser1 *fakes.FakeWriteCloser
			var remoteArtifact1 *fakes.FakeBackupArtifact

			BeforeEach(func() {
				writeCloser1 = new(fakes.FakeWriteCloser)
				remoteArtifact1 = new(fakes.FakeBackupArtifact)

				artifact.CreateArtifactReturns(writeCloser1, nil)

				instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{remoteArtifact1})

				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1, instance2})
				remoteArtifact1.ChecksumReturns(instanceChecksum, nil)
			})

			It("succeeds", func() {
				Expect(err).To(Succeed())
			})

			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateArtifactCallCount()).To(Equal(1))
				Expect(artifact.CreateArtifactArgsForCall(0)).To(Equal(remoteArtifact1))
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
					backupArtifact = new(fakes.FakeBackupArtifact)

					deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1})
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					backupArtifact.StreamFromRemoteReturns(drainError)
				})

				It("fails the transfer process", func() {
					Expect(err).To(MatchError(ContainSubstring("please make it stop")))
				})
			})

			Context("fails if file cannot be created", func() {
				var fileError = fmt.Errorf("not a good file")
				BeforeEach(func() {
					deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1})
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})
					artifact.CreateArtifactReturns(nil, fileError)
				})

				It("fails the backup process", func() {
					Expect(err).To(MatchError(ContainSubstring("not a good file")))
				})
			})

			Context("fails if local shasum calculation fails", func() {
				shasumError := fmt.Errorf("yuuuge")
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1})
					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					artifact.CreateArtifactReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(nil, shasumError)
				})

				It("fails the backup process", func() {
					Expect(err).To(MatchError(ContainSubstring("yuuuge")))
				})
			})

			Context("fails if the remote shasum cant be calulated", func() {
				remoteShasumError := fmt.Errorf("this shasum is not happy")
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1})

					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})
					backupArtifact.ChecksumReturns(nil, remoteShasumError)

					artifact.CreateArtifactReturns(writeCloser1, nil)
				})

				It("fails the backup process", func() {
					Expect(err).To(MatchError(ContainSubstring("this shasum is not happy")))
				})

				It("dosen't try to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})

			Context("fails if the remote shasum dosen't match the local shasum", func() {
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1})

					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					artifact.CreateArtifactReturns(writeCloser1, nil)

					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this won't match"}, nil)
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"file": "this wont match"}, nil)
				})

				It("fails the backup process", func() {
					Expect(err).To(MatchError(ContainSubstring("Backup is corrupted")))
					Expect(err).To(MatchError(ContainSubstring("checksums don't match for [file]")))
					Expect(err).To(MatchError(ContainSubstring("Checksum failed for 1 files in total")))
				})

				It("doesn't  to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})

			Context("fails if the number of files in the artifact dont match", func() {
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1})

					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					artifact.CreateArtifactReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"file": "this will match"}, nil)
				})

				It("fails the backup process", func() {
					Expect(err).To(MatchError(ContainSubstring("Backup is corrupted")))
				})

				It("dosen't try to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})

			Context("fails if unable to delete artifacts", func() {
				var writeCloser1 *fakes.FakeWriteCloser
				var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
				var expectedError = fmt.Errorf("unable to delete file error")

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1})

					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					artifact.CreateArtifactReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					artifact.CalculateChecksumReturns(instanceChecksum, nil)
					backupArtifact.ChecksumReturns(instanceChecksum, nil)

					backupArtifact.DeleteReturns(expectedError)
				})

				It("fails the backup process", func() {
					Expect(err).To(MatchError(ContainSubstring("unable to delete file error")))
				})
			})
		})

		Context("Many instances, failed checksum in one backable instance", func() {
			var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writer1 *fakes.FakeWriteCloser
			var writer2 *fakes.FakeWriteCloser

			var artifact1 *fakes.FakeBackupArtifact
			var artifact2 *fakes.FakeBackupArtifact

			BeforeEach(func() {
				writer1 = new(fakes.FakeWriteCloser)
				writer2 = new(fakes.FakeWriteCloser)
				artifact1 = new(fakes.FakeBackupArtifact)
				artifact2 = new(fakes.FakeBackupArtifact)

				artifact.CreateArtifactStub = func(i orchestrator.ArtifactIdentifier) (io.WriteCloser, error) {
					if i == artifact1 {
						return writer1, nil
					} else {
						return writer2, nil
					}
				}

				instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{artifact1})
				instance2.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{artifact2})

				artifact2.InstanceNameReturns("instance2")
				artifact2.InstanceIDReturns("0")

				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				artifact2.NameReturns("fixture_backup_artifact")

				deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1, instance2})
				artifact1.ChecksumReturns(instanceChecksum, nil)
				artifact2.ChecksumReturns(orchestrator.BackupChecksum{"file1": "abcd", "file2": "DOES_NOT_MATCH"}, nil)
			})

			It("fails", func() {
				By("creating artifact files on each instance", func() {
					Expect(artifact.CreateArtifactCallCount()).To(Equal(2))
					Expect(artifact.CreateArtifactArgsForCall(0)).To(Equal(artifact1))
					Expect(artifact.CreateArtifactArgsForCall(1)).To(Equal(artifact2))
				})

				By("streaming the artifact files from each instance using the writers", func() {
					Expect(artifact1.StreamFromRemoteCallCount()).To(Equal(1))
					Expect(artifact1.StreamFromRemoteArgsForCall(0)).To(Equal(writer1))

					Expect(artifact2.StreamFromRemoteCallCount()).To(Equal(1))
					Expect(artifact2.StreamFromRemoteArgsForCall(0)).To(Equal(writer2))
				})

				By("closing each writer after it has been streamed", func() {
					Expect(writer1.CloseCallCount()).To(Equal(1))
					Expect(writer2.CloseCallCount()).To(Equal(1))
				})

				By("calculating checksum for the artifact on each instance", func() {
					Expect(artifact.CalculateChecksumCallCount()).To(Equal(2))

					checksummedArtifacts := []orchestrator.ArtifactIdentifier{
						artifact.CalculateChecksumArgsForCall(0),
						artifact.CalculateChecksumArgsForCall(1),
					}

					Expect(checksummedArtifacts).To(ConsistOf(artifact1, artifact2))
				})

				By("calculating checksum for the instance on remote", func() {
					Expect(artifact1.ChecksumCallCount()).To(Equal(1))
					Expect(artifact2.ChecksumCallCount()).To(Equal(1))
				})

				By("only deleting the artifact when the checksum matches", func() {
					Expect(artifact1.DeleteCallCount()).To(Equal(1))
					Expect(artifact2.DeleteCallCount()).To(Equal(0))
				})

				By("only appending the checksum for the instance when the checksum matches", func() {
					Expect(artifact.AddChecksumCallCount()).To(Equal(1))
					actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
					Expect(actualRemoteArtifact).To(Equal(artifact1))
					Expect(acutalChecksum).To(Equal(instanceChecksum))
				})

				By("failing the backup process", func() {
					Expect(err).To(MatchError(ContainSubstring(
						"Backup is corrupted")))
					Expect(err).To(MatchError(ContainSubstring(
						"instance2/0 fixture_backup_artifact - checksums don't match for [file2]")))
					Expect(err).To(MatchError(Not(ContainSubstring("file1"))))
				})

			})

		})
	})

	Context("UploadBackupToDeployment", func() {
		var (
			artifact       *fakes.FakeBackup
			backupArtifact *fakes.FakeBackupArtifact
			err            error
			reader         io.ReadCloser
		)

		BeforeEach(func() {
			reader = ioutil.NopCloser(bytes.NewBufferString("this-is-some-backup-data"))
			deployment.RestorableInstancesReturns([]orchestrator.Instance{instance1})
			artifact = new(fakes.FakeBackup)
			artifact.ReadArtifactReturns(reader, nil)
			backupArtifact = new(fakes.FakeBackupArtifact)
		})

		JustBeforeEach(func() {
			err = artifactCopier.UploadBackupToDeployment(artifact, deployment)
		})

		Context("Single instance, restorable", func() {
			var checkSums = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				artifact.FetchChecksumReturns(checkSums, nil)
				backupArtifact.ChecksumReturns(checkSums, nil)
				instance1.ArtifactsToRestoreReturns([]orchestrator.BackupArtifact{backupArtifact})
			})

			It("does not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("checks the remote after transfer", func() {
				Expect(backupArtifact.ChecksumCallCount()).To(Equal(1))
			})

			It("marks the artifact directory as created after transfer", func() {
				Expect(instance1.MarkArtifactDirCreatedCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(1))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupArtifact))
			})

			It("logs when the copy starts and finishes", func() {
				Expect(logger.InfoCallCount()).To(Equal(2))
				_, logLine, _ := logger.InfoArgsForCall(0)
				Expect(logLine).To(ContainSubstring("Copying backup"))

				_, logLine, _ = logger.InfoArgsForCall(1)
				Expect(logLine).To(ContainSubstring("Finished copying backup"))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupArtifact.StreamToRemoteCallCount()).To(Equal(1))
				expectedReader := backupArtifact.StreamToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupArtifact.StreamToRemoteReturns(fmt.Errorf("streaming had a problem"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("streaming had a problem")))
				})
			})

			Context("problem calculating shasum on local", func() {
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, fmt.Errorf("I am so clever"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("I am so clever")))
				})
			})

			Context("problem calculating shasum on remote", func() {
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(nil, fmt.Errorf("grr"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("grr")))
				})
			})

			Context("shas don't match after transfer", func() {
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("Backup couldn't be transferred, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadArtifactReturns(nil, fmt.Errorf("leave me alone"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("leave me alone")))
				})
			})
		})

		Context("multiple instances, one restorable", func() {
			var checkSums = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				artifact.FetchChecksumReturns(checkSums, nil)
				backupArtifact.ChecksumReturns(checkSums, nil)
				instance1.ArtifactsToRestoreReturns([]orchestrator.BackupArtifact{backupArtifact})
			})

			It("does not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not ask artifacts for the non restorable instance", func() {
				Expect(instance2.ArtifactsToRestoreCallCount()).To(BeZero())
			})

			It("checks the remote after transfer", func() {
				Expect(backupArtifact.ChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(1))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupArtifact))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupArtifact.StreamToRemoteCallCount()).To(Equal(1))
				expectedReader := backupArtifact.StreamToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupArtifact.StreamToRemoteReturns(fmt.Errorf("I'm still here"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("I'm still here")))
				})
			})

			Context("problem calculating shasum on local", func() {
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, fmt.Errorf("oh well"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("oh well")))
				})
			})

			Context("problem calculating shasum on remote", func() {
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(nil, fmt.Errorf("grr"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("grr")))
				})
			})

			Context("shas don't match after transfer", func() {
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"shas": "they don't match"}, nil)
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("Backup couldn't be transferred, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadArtifactReturns(nil, fmt.Errorf("foo bar baz read error"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("foo bar baz read error")))
				})
			})
		})

		Context("Single instance, restorable, with multiple artifacts", func() {
			var checkSums = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var anotherBackupArtifact *fakes.FakeBackupArtifact

			BeforeEach(func() {
				anotherBackupArtifact = new(fakes.FakeBackupArtifact)
				artifact.FetchChecksumReturns(checkSums, nil)
				backupArtifact.ChecksumReturns(checkSums, nil)
				anotherBackupArtifact.ChecksumReturns(checkSums, nil)

				instance1.ArtifactsToRestoreReturns([]orchestrator.BackupArtifact{backupArtifact, anotherBackupArtifact})
			})

			It("does not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("checks the remote after transfer", func() {
				Expect(backupArtifact.ChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(2))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupArtifact))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(anotherBackupArtifact))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupArtifact.StreamToRemoteCallCount()).To(Equal(1))
				expectedReader := backupArtifact.StreamToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupArtifact.StreamToRemoteReturns(fmt.Errorf("this is a problem"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("this is a problem")))
				})
			})

			Context("problem calculating shasum on local", func() {
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, fmt.Errorf("checksum error occurred"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("checksum error occurred")))
				})
			})

			Context("problem calculating shasum on remote", func() {
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(nil, fmt.Errorf("grr"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("grr")))
				})
			})

			Context("shas don't match after transfer", func() {
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"file1": "abcd", "file2": "thisdoesnotmatch"}, nil)
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

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadArtifactReturns(nil, fmt.Errorf("a huge problem"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("a huge problem")))
				})
			})
		})
	})
})
