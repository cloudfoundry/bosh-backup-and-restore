package instance_test

import (
	"bytes"
	"fmt"
	"log"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/pcf-backup-and-restore/instance"
	"github.com/pivotal-cf/pcf-backup-and-restore/instance/fakes"
	backuperfakes "github.com/pivotal-cf/pcf-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("DefaultBlob", func() {

	var sshConnection *fakes.FakeSSHConnection
	var boshLogger boshlog.Logger
	var fakeInstance *backuperfakes.FakeInstance
	var stdout, stderr *gbytes.Buffer

	var defaultBlob *instance.DefaultBlob

	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		fakeInstance = new(backuperfakes.FakeInstance)

		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile), log.New(stderr, "[bosh-package] ", log.Lshortfile))

	})

	JustBeforeEach(func() {
		defaultBlob = instance.NewDefaultBlob(fakeInstance, sshConnection, boshLogger)
	})

	Describe("StreamFromRemote", func() {
		var err error
		var writer = bytes.NewBufferString("dave")

		JustBeforeEach(func() {
			err = defaultBlob.StreamFromRemote(writer)
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

	Describe("StreamBackupToRemote", func() {
		var err error
		var reader = bytes.NewBufferString("dave")

		JustBeforeEach(func() {
			err = defaultBlob.StreamBackupToRemote(reader)
		})

		Describe("when successful", func() {
			It("uses the ssh connection to make the backup directory on the remote machine", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				command := sshConnection.RunArgsForCall(0)
				Expect(command).To(Equal("sudo mkdir -p /var/vcap/store/backup/"))
			})

			It("uses the ssh connection to stream files from the remote machine", func() {
				Expect(sshConnection.StreamStdinCallCount()).To(Equal(1))
				command, sentReader := sshConnection.StreamStdinArgsForCall(0)
				Expect(command).To(Equal("sudo sh -c 'tar -C /var/vcap/store/backup -zx'"))
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

	Describe("Delete", func() {
		var err error

		JustBeforeEach(func() {
			err = defaultBlob.Delete()
		})

		It("succeeds", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("does nothing", func() {
			Expect(sshConnection.RunCallCount()).To(Equal(0))
		})
	})

})
