package bosh_test

import (
	"bytes"
	"fmt"
	"log"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	backuperfakes "github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh/fakes"
)

var _ = Describe("DefaultRemoteArtifact", func() {

	var sshConnection *fakes.FakeSSHConnection
	var boshLogger boshlog.Logger
	var instance *backuperfakes.FakeInstance
	var stdout, stderr *gbytes.Buffer

	var defaultRemoteArtifact *bosh.DefaultRemoteArtifact

	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		instance = new(backuperfakes.FakeInstance)

		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile), log.New(stderr, "[bosh-package] ", log.Lshortfile))

	})

	JustBeforeEach(func() {
		defaultRemoteArtifact = bosh.NewDefaultRemoteArtifact(instance, sshConnection, boshLogger)
	})

	Describe("StreamFromRemote", func() {
		var err error
		var writer = bytes.NewBufferString("dave")

		JustBeforeEach(func() {
			err = defaultRemoteArtifact.StreamFromRemote(writer)
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

})
