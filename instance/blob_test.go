package instance_test

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance/fakes"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	backuperfakes "github.com/pivotal-cf/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("blob", func() {

	var sshConnection *fakes.FakeSSHConnection
	var boshLogger boshlog.Logger
	var instanceToBackup *backuperfakes.FakeInstance
	var stdout, stderr *gbytes.Buffer
	var job instance.Job

	var blob orchestrator.BackupBlob

	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		instanceToBackup = new(backuperfakes.FakeInstance)
		instanceToBackup.NameReturns("redis")
		instanceToBackup.IDReturns("foo")
		instanceToBackup.IndexReturns("redis-index-1")

		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile), log.New(stderr, "[bosh-package] ", log.Lshortfile))

	})
	var BlobBehaviourForDirectory = func(blobDirectory string) {
		Describe("StreamFromRemote", func() {
			var err error
			var writer = bytes.NewBufferString("dave")

			JustBeforeEach(func() {
				err = blob.StreamFromRemote(writer)
			})

			Describe("when successful", func() {
				BeforeEach(func() {
					sshConnection.StreamReturns([]byte("not relevant"), 0, nil)
				})

				It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
					Expect(sshConnection.StreamCallCount()).To(Equal(1))

					cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
					Expect(cmd).To(Equal("sudo tar -C " + blobDirectory + " -zc ."))
					Expect(returnedWriter).To(Equal(writer))
				})

				It("does not fail", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Describe("when there is an error tarring the backup", func() {
				BeforeEach(func() {
					sshConnection.StreamReturns([]byte("not relevant"), 1, nil)
				})

				It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
					Expect(sshConnection.StreamCallCount()).To(Equal(1))

					cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
					Expect(cmd).To(Equal("sudo tar -C " + blobDirectory + " -zc ."))
					Expect(returnedWriter).To(Equal(writer))
				})

				It("fails", func() {
					Expect(err).To(HaveOccurred())
				})
			})

			Describe("when there is an SSH error", func() {
				var sshError error

				BeforeEach(func() {
					sshError = fmt.Errorf("SHH causing problems here")
					sshConnection.StreamReturns([]byte("not relevant"), 0, sshError)
				})

				It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
					Expect(sshConnection.StreamCallCount()).To(Equal(1))

					cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
					Expect(cmd).To(Equal("sudo tar -C " + blobDirectory + " -zc ."))
					Expect(returnedWriter).To(Equal(writer))
				})

				It("fails", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(sshError))
				})
			})
		})

		Describe("BackupChecksum", func() {
			var actualChecksum map[string]string
			var actualChecksumError error

			JustBeforeEach(func() {
				actualChecksum, actualChecksumError = blob.Checksum()
			})

			Context("triggers find & shasum as root", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("not relevant"), nil, 0, nil)
				})

				It("generates the correct request", func() {
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("cd " + blobDirectory + "; sudo sh -c 'find . -type f | xargs shasum'"))
				})
			})
			Context("can calculate checksum", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e  file1\n07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e  file2\nn87fc29fb3aacd99f7f7b81df9c43b13e71c56a1e file3/file4"), nil, 0, nil)
				})
				It("converts the checksum to a map", func() {
					Expect(actualChecksumError).NotTo(HaveOccurred())
					Expect(actualChecksum).To(Equal(map[string]string{
						"file1":       "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
						"file2":       "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
						"file3/file4": "n87fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
					}))
				})
			})
			Context("can calculate checksum, with trailing spaces", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e file1\n"), nil, 0, nil)
				})
				It("converts the checksum to a map", func() {
					Expect(actualChecksumError).NotTo(HaveOccurred())
					Expect(actualChecksum).To(Equal(map[string]string{
						"file1": "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
					}))
				})
			})
			Context("sha output is empty", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte(""), nil, 0, nil)
				})
				It("converts an empty map", func() {
					Expect(actualChecksumError).NotTo(HaveOccurred())
					Expect(actualChecksum).To(Equal(map[string]string{}))
				})
			})
			Context("sha for a empty directory", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("da39a3ee5e6b4b0d3255bfef95601890afd80709  -"), nil, 0, nil)
				})
				It("reject '-' as a filename", func() {
					Expect(actualChecksumError).NotTo(HaveOccurred())
					Expect(actualChecksum).To(Equal(map[string]string{}))
				})
			})

			Context("fails to calculate checksum", func() {
				expectedErr := fmt.Errorf("some error")

				BeforeEach(func() {
					sshConnection.RunReturns(nil, nil, 0, expectedErr)
				})
				It("returns an error", func() {
					Expect(actualChecksumError).To(MatchError(expectedErr))
				})
			})
			Context("fails to execute the command", func() {
				BeforeEach(func() {
					sshConnection.RunReturns(nil, nil, 1, nil)
				})
				It("returns an error", func() {
					Expect(actualChecksumError).To(HaveOccurred())
				})
			})
		})

		Describe("Delete", func() {
			var err error

			JustBeforeEach(func() {
				err = blob.Delete()
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("deletes only the named blob's backup directory on the remote", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo rm -rf " + blobDirectory))
			})

			Context("when there is an error with the SSH connection", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = fmt.Errorf("you fool")
					sshConnection.RunReturns([]byte("don't matter"), []byte("don't matter"), 0, expectedErr)
				})

				It("fails", func() {
					Expect(err).To(MatchError(expectedErr))
				})
			})

			Context("when the rm command returns an error", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("don't matter"), []byte("don't matter"), 1, nil)
				})

				It("fails", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						"Error deleting blobs on instance redis/foo. Directory name " + blobDirectory + ". Exit code 1",
					))
				})
			})
		})

		Describe("BackupSize", func() {
			Context("when there is a backup", func() {
				var size string

				BeforeEach(func() {
					sshConnection.RunReturns([]byte("4.1G\n"), nil, 0, nil)
				})

				JustBeforeEach(func() {
					size, _ = blob.Size()
				})

				It("returns the size of the backup according to the root user, as a string", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo du -sh " + blobDirectory + " | cut -f1"))
					Expect(size).To(Equal("4.1G"))
				})
			})

			Context("when there is no backup directory", func() {
				var err error

				BeforeEach(func() {
					sshConnection.RunReturns(nil, nil, 1, nil) // simulating file not found
				})

				JustBeforeEach(func() {
					_, err = blob.Size()
				})

				It("returns the size of the backup according to the root user, as a string", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo du -sh " + blobDirectory + " | cut -f1"))
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when an error occurs", func() {
				var err error
				var actualError = errors.New("oh noes, more errors")

				BeforeEach(func() {
					sshConnection.RunReturns(nil, nil, 0, actualError)
				})

				JustBeforeEach(func() {
					_, err = blob.Size()
				})

				It("returns the error", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					Expect(err).To(MatchError(actualError))
				})
			})

		})

		Describe("StreamBackupToRemote", func() {
			var err error
			var reader = bytes.NewBufferString("dave")

			JustBeforeEach(func() {
				err = blob.StreamToRemote(reader)
			})

			Describe("when successful", func() {
				It("uses the ssh connection to make the backup directory on the remote machine", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					command := sshConnection.RunArgsForCall(0)
					Expect(command).To(Equal("sudo mkdir -p " + blobDirectory))
				})

				It("uses the ssh connection to stream files from the remote machine", func() {
					Expect(sshConnection.StreamStdinCallCount()).To(Equal(1))
					command, sentReader := sshConnection.StreamStdinArgsForCall(0)
					Expect(command).To(Equal("sudo sh -c 'tar -C " + blobDirectory + " -zx'"))
					Expect(reader).To(Equal(sentReader))
				})

				It("does not fail", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Describe("when the remote side returns an error", func() {
				BeforeEach(func() {
					sshConnection.StreamStdinReturns([]byte("not relevant"), []byte("All the pies"), 1, nil)
				})

				It("fails and return the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("All the pies"))
				})
			})

			Describe("when there is an error running the stream", func() {
				BeforeEach(func() {
					sshConnection.StreamStdinReturns([]byte("not relevant"), []byte("not relevant"), 0, fmt.Errorf("Errorerrororororororor"))
				})

				It("fails", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Errorerrororororororor"))
				})
			})

			Describe("when creating the directory fails on the remote", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 1, nil)
				})

				It("fails and returns the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Creating backup directory on the remote returned 1"))
				})
			})

			Describe("when creating the directory fails because of a connection error", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, fmt.Errorf("I refuse to create you this directory."))
				})

				It("fails and returns the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("I refuse to create you this directory."))
				})
			})
		})
	}

	Context("NamedBackupBlob", func() {
		BeforeEach(func() {
			job = instance.NewJob(instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/b-backup"}, instance.Metadata{BackupName: "named-blob"})
		})
		JustBeforeEach(func() {
			blob = instance.NewNamedBackupBlob(instanceToBackup, job, sshConnection, boshLogger)
		})

		Describe("Name", func() {
			It("returns the blob", func() {
				Expect(blob.Name()).To(Equal("named-blob"))
			})
		})

		Describe("ID", func() {
			BeforeEach(func() {
				instanceToBackup.IDReturns("instance-id")
			})
			It("returns instances id", func() {
				Expect(blob.ID()).To(Equal("instance-id"))
			})
		})

		Describe("Index", func() {
			It("returns blank", func() {
				Expect(blob.Index()).To(BeEmpty())
			})
		})

		Describe("IsNamed", func() {
			It("returns true", func() {
				Expect(blob.IsNamed()).To(BeTrue())
			})
		})
		BlobBehaviourForDirectory("/var/vcap/store/backup/named-blob")
	})

	Context("NamedRestoreBlob", func() {
		BeforeEach(func() {
			job = instance.NewJob(instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/b-restore"}, instance.Metadata{RestoreName: "named-blob-to-restore"})
		})
		JustBeforeEach(func() {
			blob = instance.NewNamedRestoreBlob(instanceToBackup, job, sshConnection, boshLogger)
		})

		Describe("Name", func() {
			It("returns the blob", func() {
				Expect(blob.Name()).To(Equal("named-blob-to-restore"))
			})
		})

		Describe("ID", func() {
			BeforeEach(func() {
				instanceToBackup.IDReturns("instance-id")
			})
			It("returns instances id", func() {
				Expect(blob.ID()).To(Equal("instance-id"))
			})
		})

		Describe("Index", func() {
			It("returns blank", func() {
				Expect(blob.Index()).To(BeEmpty())
			})
		})

		Describe("IsNamed", func() {
			It("returns true", func() {
				Expect(blob.IsNamed()).To(BeTrue())
			})
		})
		BlobBehaviourForDirectory("/var/vcap/store/backup/named-blob-to-restore")
	})

	Context("DefaultBlob", func() {
		BeforeEach(func() {
			job = instance.NewJob(instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/b-restore"}, instance.Metadata{RestoreName: "named-blob-to-restore"})
		})
		JustBeforeEach(func() {
			blob = instance.NewDefaultBlob(instanceToBackup, sshConnection, boshLogger)
		})

		Describe("Name", func() {
			It("returns the blob", func() {
				Expect(blob.Name()).To(Equal("redis"))
			})
		})

		Describe("ID", func() {
			BeforeEach(func() {
				instanceToBackup.IDReturns("instance-id")
			})
			It("returns instances id", func() {
				Expect(blob.ID()).To(Equal("instance-id"))
			})
		})

		Describe("Index", func() {
			It("returns instances index", func() {
				Expect(blob.Index()).To(Equal("redis-index-1"))
			})
		})

		Describe("IsNamed", func() {
			It("returns false", func() {
				Expect(blob.IsNamed()).To(BeFalse())
			})
		})
		BlobBehaviourForDirectory("/var/vcap/store/backup")
	})

})
