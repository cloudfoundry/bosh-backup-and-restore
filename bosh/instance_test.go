package bosh_test

import (
	"bytes"
	"fmt"
	"log"

	"github.com/cloudfoundry/bosh-cli/director"
	boshfakes "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh/fakes"
)

var _ = Describe("Instance", func() {
	var sshConnection *fakes.FakeSSHConnection
	var boshDeployment *boshfakes.FakeDeployment
	var boshLogger boshlog.Logger

	var instance backuper.Instance
	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		boshDeployment = new(boshfakes.FakeDeployment)
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(GinkgoWriter, "[bosh-package] ", log.Lshortfile), log.New(GinkgoWriter, "[bosh-package] ", log.Lshortfile))
	})

	JustBeforeEach(func() {
		sshConnection.UsernameReturns("sshUsername")
		instance = bosh.NewBoshInstance("job-name", "job-index", sshConnection, boshDeployment, boshLogger)
	})

	Context("IsBackupable", func() {
		var actualBackupable bool
		var actualError error

		JustBeforeEach(func() {
			actualBackupable, actualError = instance.IsBackupable()
		})

		Describe("there are backup scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, nil)
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
			It("returns true", func() {
				Expect(actualBackupable).To(BeTrue())
			})
			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/backup"))
			})
		})

		Describe("there are no backup scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 1, nil)
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
			It("returns false", func() {
				Expect(actualBackupable).To(BeFalse())
			})
			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/backup"))
			})
		})

		Describe("error while running command", func() {
			var expectedError = fmt.Errorf("we need to build a wall")
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, expectedError)
			})
			It("succeeds", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/backup"))
			})
		})
	})

	Context("IsRestorable", func() {
		var actualRestorable bool
		var actualError error

		JustBeforeEach(func() {
			actualRestorable, actualError = instance.IsRestorable()
		})

		Describe("there are restore scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, nil)
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
			It("returns true", func() {
				Expect(actualRestorable).To(BeTrue())
			})
			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/restore"))
			})
		})

		Describe("there are no restore scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 1, nil)
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
			It("returns false", func() {
				Expect(actualRestorable).To(BeFalse())
			})
			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/restore"))
			})
		})

		Describe("error while running command", func() {
			var expectedError = fmt.Errorf("we need to build a wall")
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, expectedError)
			})
			It("succeeds", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/restore"))
			})
		})
	})

	Context("Backup", func() {
		var err error

		JustBeforeEach(func() {
			err = instance.Backup()
		})
		Describe("when there are backup scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, nil)
			})
			It("uses the ssh connection to create the backup dir and run all backup scripts as sudo", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo mkdir -p /var/vcap/store/backup && ls /var/vcap/jobs/*/bin/backup | xargs -IN sudo sh -c N"))
			})
			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("when there is an error backing up", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 1, nil)
			})
			It("uses the ssh connection to create the backup dir and run all backup scripts as sudo", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo mkdir -p /var/vcap/store/backup && ls /var/vcap/jobs/*/bin/backup | xargs -IN sudo sh -c N"))
			})
			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("Restore", func() {
		var actualError error

		JustBeforeEach(func() {
			actualError = instance.Restore()
		})

		Describe("runs restore successfully", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, nil)
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(2))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("cd /var/vcap/store/backup && sudo tar -zxvf backup.tgz"))
				Expect(sshConnection.RunArgsForCall(1)).To(Equal("ls /var/vcap/jobs/*/bin/restore | xargs -IN sudo sh -c N"))
			})
		})

		Describe("error while running command", func() {
			var expectedError = fmt.Errorf("we need to build a wall")
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, expectedError)
			})
			It("returns error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		Describe("restore scripts return an error", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("my fingers are long and beautiful"), 1, nil)
			})
			It("returns error", func() {
				Expect(actualError.Error()).To(ContainSubstring("Instance restore scripts returned %d. Error: %s", 1, "my fingers are long and beautiful"))
			})
		})
	})

	Context("StreamBackupToRemote", func() {
		var err error
		var reader = bytes.NewBufferString("dave")

		JustBeforeEach(func() {
			err = instance.StreamBackupToRemote(reader)
		})

		Describe("when successful", func() {
			It("uses the ssh connection to make the backup directory on the remote machine", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				command := sshConnection.RunArgsForCall(0)
				Expect(command).To(Equal("sudo mkdir -p /var/vcap/store/backup/; sudo chown vcap:vcap /var/vcap/store/backup"))
			})

			It("uses the ssh connection to stream files from the remote machine", func() {
				Expect(sshConnection.StreamStdinCallCount()).To(Equal(1))
				command, sentReader := sshConnection.StreamStdinArgsForCall(0)
				Expect(command).To(Equal("sudo -i -u vcap bash -c 'cat > /var/vcap/store/backup/backup.tgz'"))
				Expect(reader).To(Equal(sentReader))
			})

			It("does not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("when the remote side returns an error", func() {
			BeforeEach(func() {
				sshConnection.StreamStdinReturns([]byte("not relevant"), []byte("The beauty of me is that I’m very rich."), 1, nil)
			})

			It("fails and return the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("The beauty of me is that I’m very rich."))
			})
		})

		Describe("when there is an error running the stream", func() {
			BeforeEach(func() {
				sshConnection.StreamStdinReturns([]byte("not relevant"), []byte("not relevant"), 0, fmt.Errorf("My Twitter has become so powerful"))
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("My Twitter has become so powerful"))
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
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, fmt.Errorf("These media people. The most dishonest people"))
			})

			It("fails and returns the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("These media people. The most dishonest people"))
			})
		})
	})

	Context("StreamBackupFromRemote", func() {
		var err error
		var writer = bytes.NewBufferString("dave")

		JustBeforeEach(func() {
			err = instance.StreamBackupFromRemote(writer)
		})

		Describe("when successful", func() {
			BeforeEach(func() {
				sshConnection.StreamReturns([]byte("not relevant"), 0, nil)
			})

			It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
				Expect(sshConnection.StreamCallCount()).To(Equal(1))

				cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
				Expect(cmd).To(Equal("sudo tar -C /var/vcap/store/backup -zc ."))
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
				Expect(cmd).To(Equal("sudo tar -C /var/vcap/store/backup -zc ."))
				Expect(returnedWriter).To(Equal(writer))
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("when there is an SSH error", func() {
			var sshError error

			BeforeEach(func() {
				sshError = fmt.Errorf("I have the best SSH")
				sshConnection.StreamReturns([]byte("not relevant"), 0, sshError)
			})

			It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
				Expect(sshConnection.StreamCallCount()).To(Equal(1))

				cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
				Expect(cmd).To(Equal("sudo tar -C /var/vcap/store/backup -zc ."))
				Expect(returnedWriter).To(Equal(writer))
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(sshError))
			})
		})
	})

	Context("Cleanup", func() {
		var actualError error
		var expectedError error

		JustBeforeEach(func() {
			actualError = instance.Cleanup()
		})
		Describe("cleans up successfully", func() {
			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job-name", "job-index")))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})
		Describe("error while running delete", func() {
			BeforeEach(func() {
				expectedError = fmt.Errorf("werk niet")
				boshDeployment.CleanUpSSHReturns(expectedError)
			})
			It("fails", func() {
				Expect(actualError).To(MatchError(expectedError))
			})
		})
	})

	Context("BackupChecksum", func() {
		var actualChecksum map[string]string
		var actualChecksumError error
		JustBeforeEach(func() {
			actualChecksum, actualChecksumError = instance.BackupChecksum()
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

	Context("Name", func() {
		It("returns the instance name", func() {
			Expect(instance.Name()).To(Equal("job-name"))
		})
	})

	Context("ID", func() {
		It("returns the instance ID", func() {
			Expect(instance.ID()).To(Equal("job-index"))
		})
	})

	Context("BackupSize", func() {
		Context("when there is a backup", func() {
			var size string

			BeforeEach(func() {
				sshConnection.RunReturns([]byte("4.1G\n"), nil, 0, nil)
			})

			JustBeforeEach(func() {
				size, _ = instance.BackupSize()
			})

			It("returns the size of the backup, as a string", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("du -sh /var/vcap/store/backup/ | cut -f1"))
				Expect(size).To(Equal("4.1G"))
			})
		})

		Context("when there is no backup directory", func() {
			var err error

			BeforeEach(func() {
				sshConnection.RunReturns(nil, nil, 1, nil) // simulating file not found
			})

			JustBeforeEach(func() {
				_, err = instance.BackupSize()
			})

			It("returns the size of the backup, as a string", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("du -sh /var/vcap/store/backup/ | cut -f1"))
				Expect(err).To(HaveOccurred())
			})
		})

	})
})
