package orchestrator_test

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

	"bytes"
	"fmt"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"io/ioutil"
)

var _ = Describe("BackupUploadExecutable", func() {
	var (
		executable     executor.Executable
		backup         *fakes.FakeBackup
		remoteArtifact *fakes.FakeBackupArtifact
		instance       *fakes.FakeInstance
		logger         *fakes.FakeLogger
		actualError    error
		localBackupArtifactReader io.ReadCloser
	)
	BeforeEach(func() {
		backup = new(fakes.FakeBackup)
		remoteArtifact = new(fakes.FakeBackupArtifact)
		instance = new(fakes.FakeInstance)
		logger = new(fakes.FakeLogger)


		localBackupArtifactReader = ioutil.NopCloser(bytes.NewBufferString("this-is-some-backup-data"))
		backup.ReadArtifactReturns(localBackupArtifactReader, nil)
		backup.FetchChecksumReturns(orchestrator.BackupChecksum{"file1": "abcd", "file2": "foo"}, nil)
		remoteArtifact.ChecksumReturns(orchestrator.BackupChecksum{"file1": "abcd", "file2": "foo"}, nil)
	})

	JustBeforeEach(func() {
		executable = orchestrator.NewBackupUploadExecutable(backup, remoteArtifact, instance, logger)
		actualError = executable.Execute()

	})

	It("uploads the backup", func() {
		By("not failing", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		By("fetching the remote artifact from the backup", func() {
			Expect(backup.ReadArtifactCallCount()).To(Equal(1))
			Expect(backup.ReadArtifactArgsForCall(0)).To(Equal(remoteArtifact))
		})

		By("streaming local artifact to remote", func() {
			Expect(remoteArtifact.StreamToRemoteCallCount()).To(Equal(1))
			Expect(remoteArtifact.StreamToRemoteArgsForCall(0)).To(Equal(localBackupArtifactReader))
		})

		By("marking the director created", func() {
			Expect(instance.MarkArtifactDirCreatedCallCount()).To(Equal(1))
		})

		By("fetching local checksum", func() {
			Expect(backup.ReadArtifactCallCount()).To(Equal(1))
			Expect(backup.FetchChecksumArgsForCall(0)).To(Equal(remoteArtifact))
		})

		By("calculating the remote checksum", func() {
			Expect(remoteArtifact.ChecksumCallCount()).To(Equal(1))
		})

		By("logging the upload", func() {
			Expect(logger.InfoCallCount()).To(BeNumerically(">", 0))
		})
	})

	Context("When the artifact cannot be read from the backup", func() {
		BeforeEach(func() {
			backup.ReadArtifactReturns(nil, fmt.Errorf("artifact error"))
		})

		It("should fail", func() {
			Expect(actualError).To(MatchError("artifact error"))
		})
	})

	Context("When the local artifact cannot be streamed to remote", func() {
		BeforeEach(func() {
			remoteArtifact.StreamToRemoteReturns(fmt.Errorf("stream error"))
		})

		It("should fail", func() {
			Expect(actualError).To(MatchError("stream error"))
		})
	})

	Context("When the checksum cannot be fetched from the localbackup", func() {
		BeforeEach(func() {
			backup.FetchChecksumReturns(nil, fmt.Errorf("checksum error"))
		})

		It("should fail", func() {
			Expect(actualError).To(MatchError("checksum error"))
		})
	})

	Context("When the checksums are mismatched", func() {
		BeforeEach(func() {
			remoteArtifact.ChecksumReturns(orchestrator.BackupChecksum{"file1": "abcd", "file2": "not matching"}, nil)
		})

		It("should fail", func() {
			Expect(actualError).To(MatchError(ContainSubstring("Backup couldn't be transferred, checksum failed")))
		})
	})

})
