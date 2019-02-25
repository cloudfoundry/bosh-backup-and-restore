package instance_test

import (
	"bytes"
	"fmt"
	"log"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	backuperfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	sshfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("artifact", func() {
	var remoteRunner *sshfakes.FakeRemoteRunner
	var boshLogger boshlog.Logger
	var testInstance *backuperfakes.FakeInstance
	var logOutput *gbytes.Buffer
	var job instance.Job

	var backupArtifact orchestrator.BackupArtifact

	BeforeEach(func() {
		remoteRunner = new(sshfakes.FakeRemoteRunner)
		testInstance = new(backuperfakes.FakeInstance)
		testInstance.NameReturns("redis")
		testInstance.IDReturns("foo")
		testInstance.IndexReturns("redis-index-1")

		logOutput = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(logOutput, "[bosh-package] ", log.Lshortfile))
	})

	var ArtifactBehaviourForDirectory = func(artifactDirectory string) {
		Describe("StreamFromRemote", func() {
			var err error
			var writer = bytes.NewBufferString("dave")

			JustBeforeEach(func() {
				err = backupArtifact.StreamFromRemote(writer)
			})

			Describe("when successful", func() {
				BeforeEach(func() {
					remoteRunner.ArchiveAndDownloadReturns(nil)
				})

				It("uses the remote runner to tar the backup and download it to the local machine", func() {
					Expect(remoteRunner.ArchiveAndDownloadCallCount()).To(Equal(1))

					dir, returnedWriter := remoteRunner.ArchiveAndDownloadArgsForCall(0)
					Expect(dir).To(Equal(artifactDirectory))
					Expect(returnedWriter).To(Equal(writer))
				})

				It("does not fail", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Describe("when there is an error in archive and download", func() {
				BeforeEach(func() {
					remoteRunner.ArchiveAndDownloadReturns(fmt.Errorf("oh no, it broke"))
				})

				It("uses the remote runner to tar the backup and download it to the local machine", func() {
					Expect(remoteRunner.ArchiveAndDownloadCallCount()).To(Equal(1))

					dir, returnedWriter := remoteRunner.ArchiveAndDownloadArgsForCall(0)
					Expect(dir).To(Equal(artifactDirectory))
					Expect(returnedWriter).To(Equal(writer))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("oh no, it broke")))
				})
			})
		})

		Describe("BackupChecksum", func() {
			var actualChecksum map[string]string
			var actualChecksumError error

			JustBeforeEach(func() {
				actualChecksum, actualChecksumError = backupArtifact.Checksum()
			})

			Context("can calculate checksum", func() {
				checksum := map[string]string{
					"file1":       "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
					"file2":       "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
					"file3/file4": "n87fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
				}

				BeforeEach(func() {
					remoteRunner.ChecksumDirectoryReturns(checksum, nil)
				})

				It("generates the correct request", func() {
					Expect(remoteRunner.ChecksumDirectoryArgsForCall(0)).To(Equal(artifactDirectory))
				})

				It("returns the checksum", func() {
					Expect(actualChecksumError).NotTo(HaveOccurred())
					Expect(actualChecksum).To(Equal(checksum))
				})
			})

			Context("fails to calculate checksum", func() {
				BeforeEach(func() {
					remoteRunner.ChecksumDirectoryReturns(nil, fmt.Errorf("some error"))
				})

				It("returns an error", func() {
					Expect(actualChecksumError).To(MatchError(ContainSubstring("some error")))
				})
			})
		})

		Describe("Delete", func() {
			var err error

			JustBeforeEach(func() {
				err = backupArtifact.Delete()
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("deletes only the named artifact directory on the remote", func() {
				Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(1))
				Expect(remoteRunner.RemoveDirectoryArgsForCall(0)).To(Equal(artifactDirectory))
			})

			Context("when there is an error from the remote runner", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = fmt.Errorf("nope")
					remoteRunner.RemoveDirectoryReturns(expectedErr)
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("nope")))
				})
			})
		})

		Describe("BackupSize", func() {
			Context("when there is a backup", func() {
				var size string

				BeforeEach(func() {
					remoteRunner.SizeOfReturns("4.1G", nil)
				})

				JustBeforeEach(func() {
					size, _ = backupArtifact.Size()
				})

				It("returns the size of the backup according to the root user, as a string", func() {
					Expect(remoteRunner.SizeOfCallCount()).To(Equal(1))
					Expect(remoteRunner.SizeOfArgsForCall(0)).To(Equal(artifactDirectory))
					Expect(size).To(Equal("4.1G"))
				})
			})

			Context("when an error occurs", func() {
				var err error

				BeforeEach(func() {
					remoteRunner.SizeOfReturns("", fmt.Errorf("no backup directory or something")) // simulating file not found
				})

				JustBeforeEach(func() {
					_, err = backupArtifact.Size()
				})

				It("returns the size of the backup according to the root user, as a string", func() {
					Expect(remoteRunner.SizeOfCallCount()).To(Equal(1))
					Expect(remoteRunner.SizeOfArgsForCall(0)).To(Equal(artifactDirectory))
					Expect(err).To(SatisfyAll(
						MatchError(ContainSubstring("Unable to check size of "+artifactDirectory)),
						MatchError(ContainSubstring("no backup directory or something")),
					))
				})
			})
		})

		Describe("StreamBackupToRemote", func() {
			var err error
			var reader = bytes.NewBufferString("dave")

			JustBeforeEach(func() {
				err = backupArtifact.StreamToRemote(reader)
			})

			Describe("when successful", func() {
				It("uses the remote runner to make the backup directory on the remote machine", func() {
					Expect(remoteRunner.CreateDirectoryCallCount()).To(Equal(1))
					dir := remoteRunner.CreateDirectoryArgsForCall(0)
					Expect(dir).To(Equal(artifactDirectory))
				})

				It("uses the remote runner to stream files to the remote machine", func() {
					Expect(remoteRunner.ExtractAndUploadCallCount()).To(Equal(1))
					sentReader, dir := remoteRunner.ExtractAndUploadArgsForCall(0)
					Expect(dir).To(Equal(artifactDirectory))
					Expect(reader).To(Equal(sentReader))
				})

				It("does not fail", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Describe("when the remote runner returns an error for creating a directory", func() {
				BeforeEach(func() {
					remoteRunner.CreateDirectoryReturns(fmt.Errorf("I refuse to create you this directory."))
				})

				It("fails and returns the error", func() {
					Expect(err).To(MatchError(ContainSubstring("I refuse to create you this directory.")))
				})
			})

			Describe("when the remote runner returns an error for extracting and uploading a directory", func() {
				BeforeEach(func() {
					remoteRunner.ExtractAndUploadReturns(fmt.Errorf("I refuse to upload this directory."))
				})

				It("fails and returns the error", func() {
					Expect(err).To(MatchError(ContainSubstring("I refuse to upload this directory.")))
				})
			})
		})
	}

	Context("BackupArtifact", func() {
		JustBeforeEach(func() {
			backupArtifact = instance.NewBackupArtifact(job, testInstance, remoteRunner, boshLogger)
		})

		Context("Named Artifact", func() {
			BeforeEach(func() {
				job = instance.NewJob(nil, "", nil, "", instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/start_ctl"}, instance.Metadata{BackupName: "named-artifact-to-backup"}, false)
			})

			It("is named with the job's custom backup name", func() {
				Expect(backupArtifact.Name()).To(Equal(job.BackupArtifactName()))
			})

			It("has a custom name", func() {
				Expect(backupArtifact.HasCustomName()).To(BeTrue())
			})

			ArtifactBehaviourForDirectory("/var/vcap/store/bbr-backup/named-artifact-to-backup")
		})

		Context("Default Artifact", func() {
			BeforeEach(func() {
				job = instance.NewJob(nil, "", nil, "", instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/start_ctl"}, instance.Metadata{}, false)
			})

			It("is named after the job", func() {
				Expect(backupArtifact.Name()).To(Equal(job.Name()))
			})

			It("does not have a custom name", func() {
				Expect(backupArtifact.HasCustomName()).To(BeFalse())
			})

			Describe("InstanceName", func() {
				It("returns the instance name", func() {
					Expect(backupArtifact.InstanceName()).To(Equal(testInstance.Name()))
				})
			})

			Describe("InstanceIndex", func() {
				It("returns the instance index", func() {
					Expect(backupArtifact.InstanceIndex()).To(Equal(testInstance.Index()))
				})
			})

			ArtifactBehaviourForDirectory("/var/vcap/store/bbr-backup/foo1")
		})
	})

	Context("RestoreArtifact", func() {
		JustBeforeEach(func() {
			backupArtifact = instance.NewRestoreArtifact(job, testInstance, remoteRunner, boshLogger)
		})

		Context("Named Artifact", func() {
			BeforeEach(func() {
				job = instance.NewJob(nil, "", nil, "", instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/start_ctl"}, instance.Metadata{RestoreName: "named-artifact-to-restore"}, false)
			})

			It("is named with the job's custom backup name", func() {
				Expect(backupArtifact.Name()).To(Equal(job.RestoreArtifactName()))
			})

			It("has a custom name", func() {
				Expect(backupArtifact.HasCustomName()).To(BeTrue())
			})

			ArtifactBehaviourForDirectory("/var/vcap/store/bbr-backup/named-artifact-to-restore")
		})

		Context("Default Artifact", func() {
			BeforeEach(func() {
				job = instance.NewJob(nil, "", nil, "", instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/start_ctl"}, instance.Metadata{}, false)
			})

			It("is named after the job", func() {
				Expect(backupArtifact.Name()).To(Equal(job.Name()))
			})

			It("does not have a custom name", func() {
				Expect(backupArtifact.HasCustomName()).To(BeFalse())
			})

			Describe("InstanceName", func() {
				It("returns the instance name", func() {
					Expect(backupArtifact.InstanceName()).To(Equal(testInstance.Name()))
				})
			})

			Describe("InstanceIndex", func() {
				It("returns the instance index", func() {
					Expect(backupArtifact.InstanceIndex()).To(Equal(testInstance.Index()))
				})
			})

			ArtifactBehaviourForDirectory("/var/vcap/store/bbr-backup/foo1")
		})
	})
})
