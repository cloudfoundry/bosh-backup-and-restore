package instance_test

import (
	"bytes"
	"io"
	"log"

	"io/ioutil"

	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gossh "golang.org/x/crypto/ssh"
)

var privateKey string
var user string
var testInstance *testcluster.Instance
var sshConnection ssh.SSHConnection

var _ = Describe("SshRemoteRunner", func() {
	var remoteRunner instance.SshRemoteRunner

	BeforeEach(func() {
		user = "test-user"
		testInstance = testcluster.NewInstanceWithKeepAlive(2)
		testInstance.CreateUser(user, publicKeyForDocker(defaultPrivateKey))
		privateKey = defaultPrivateKey

		hostPublicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(testInstance.HostPublicKey()))
		Expect(err).NotTo(HaveOccurred())

		combinedOutLog := log.New(io.MultiWriter(GinkgoWriter, bytes.NewBufferString("")), "[bosh-package] ", log.Lshortfile)
		combinedErrLog := log.New(io.MultiWriter(GinkgoWriter, bytes.NewBufferString("")), "[bosh-package] ", log.Lshortfile)
		logger := boshlog.New(boshlog.LevelDebug, combinedOutLog, combinedErrLog)

		sshConnection, err = ssh.NewConnection(testInstance.Address(), user, privateKey, gossh.FixedHostKey(hostPublicKey),
			[]string{hostPublicKey.Type()}, logger)
		Expect(err).NotTo(HaveOccurred())

		remoteRunner = instance.NewRemoteRunner(sshConnection, logger)
	})

	AfterEach(func() {
		testInstance.DieInBackground()
	})

	var runCommand = func(cmd string) string {
		stdout, stderr, status, err := sshConnection.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(0), string(stderr))
		return string(stdout)
	}

	Describe("ConnectedUsername", func() {
		It("returns the name of the connected user", func() {
			Expect(remoteRunner.ConnectedUsername()).To(Equal(user))
		})
	})

	Describe("DirectoryExists", func() {
		Context("When the directory does not exist", func() {
			It("returns false", func() {
				Expect(remoteRunner.DirectoryExists("/tmp/non-existing-dir")).To(BeFalse())
			})
		})

		Context("When the directory exists", func() {
			BeforeEach(func() {
				runCommand("mkdir -p /tmp/an-existing-dir")
			})

			It("returns false", func() {
				Expect(remoteRunner.DirectoryExists("/tmp/an-existing-dir")).To(BeTrue())
			})
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				_, err := remoteRunner.DirectoryExists("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("CreateDirectory", func() {
		It("creates a directory", func() {
			Expect(remoteRunner.CreateDirectory("/tmp/a-new-directory")).To(Succeed())
			Expect(remoteRunner.DirectoryExists("/tmp/a-new-directory")).To(BeTrue())
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				err := remoteRunner.CreateDirectory("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("RemoveDirectory", func() {
		Context("When the directory exists", func() {
			BeforeEach(func() {
				runCommand("mkdir -p /tmp/existing-directory")
				runCommand("sudo chown root:root /tmp/existing-directory")
			})

			It("removes the directory", func() {
				err := remoteRunner.RemoveDirectory("/tmp/existing-directory")

				Expect(err).NotTo(HaveOccurred())
				Expect(remoteRunner.DirectoryExists("/tmp/non-existing-dir")).To(BeFalse())
			})
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				err := remoteRunner.RemoveDirectory("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("remote archiving and extracting of directories", func() {
		It("archives and extracts directories from/to remote servers", func() {
			runCommand("mkdir -p /tmp/dir-to-archive")
			runCommand("echo 'one' > /tmp/dir-to-archive/file1")
			runCommand("echo 'two' > /tmp/dir-to-archive/file2")

			By("downloading and archiving the directory")
			archiveFile := makeTmpFile("remote-runner-test-")
			err := remoteRunner.ArchiveAndDownload("/tmp/dir-to-archive", archiveFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(fileSize(archiveFile)).To(BeNumerically(">", 0))

			By("by extracting and uploading the archive to a specified directory")
			runCommand("mkdir -p /tmp/uploaded-dir")
			archiveFile = resetCursor(archiveFile)
			err = remoteRunner.ExtractAndUpload(archiveFile, "/tmp/uploaded-dir")
			Expect(err).NotTo(HaveOccurred())
			lsOutput := runCommand("ls /tmp/uploaded-dir")
			Expect(lsOutput).To(Equal("file1\nfile2\n"))
		})

		Context("when archiving fails", func() {
			Context("when the command fails", func() {
				It("returns an error", func() {
					archiveFile := makeTmpFile("remote-runner-test-")
					err := remoteRunner.ArchiveAndDownload("/tmp/unexisting-dir", archiveFile)
					Expect(err).To(MatchError(ContainSubstring("No such file or directory")))
				})
			})

			Context("when the connection fails", func() {
				BeforeEach(func() {
					destroyInstance(testInstance)
				})

				It("returns an error", func() {
					archiveFile := makeTmpFile("remote-runner-test-")
					err := remoteRunner.ArchiveAndDownload("/tmp/unexisting-dir", archiveFile)
					Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
				})
			})
		})

		Context("when extracting fails", func() {
			Context("when the command fails", func() {
				It("returns an error", func() {
					notATar := makeTmpFile("remote-runner-test-")
					err := remoteRunner.ExtractAndUpload(notATar, "/tmp/arbitrary-dir")
					Expect(err).To(MatchError(ContainSubstring("This does not look like a tar archive")))
				})
			})

			Context("when the connection fails", func() {
				BeforeEach(func() {
					destroyInstance(testInstance)
				})

				It("returns an error", func() {
					arbitraryFile := makeTmpFile("remote-runner-test-")
					err := remoteRunner.ExtractAndUpload(arbitraryFile, "/tmp/arbitrary-dir")
					Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
				})
			})
		})
	})

	Describe("SizeOf", func() {
		Context("when the file or directory exists", func() {
			BeforeEach(func() {
				runCommand("mkdir /tmp/a-dir")
				runCommand("dd if=/dev/zero of=/tmp/a-dir/a-file bs=1k count=1000")
				runCommand("dd if=/dev/zero of=/tmp/a-dir/b-file bs=1k count=500")
			})

			It("returns a string with the specified file or directory size", func() {
				Expect(remoteRunner.SizeOf("/tmp/a-dir")).To(Equal("1.5M"))
			})
		})

		Context("when the directory does not exist", func() {
			It("returns an error", func() {
				_, err := remoteRunner.SizeOf("/tmp/not-a-file")
				Expect(err).To(MatchError(ContainSubstring("No such file or directory")))
			})
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				_, err := remoteRunner.SizeOf("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("ChecksumDirectory", func() {
		Context("when the file or directory exists", func() {
			BeforeEach(func() {
				runCommand("mkdir /tmp/a-dir")
				runCommand("echo 'foo' > /tmp/a-dir/file1")
				runCommand("echo 'bar' > /tmp/a-dir/file2")
			})

			It("calculates the SHA256 checksum for each file in the directory", func() {
				Expect(remoteRunner.ChecksumDirectory("/tmp/a-dir")).To(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("./file1", "b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c"),
					HaveKeyWithValue("./file2", "7d865e959b2466918c9863afca942d0fb89d7c9ac0c99bafc3749504ded97730"),
				))
			})
		})

		Context("when the directory does not exist", func() {
			It("returns an error", func() {
				_, err := remoteRunner.ChecksumDirectory("/tmp/not-a-dir")
				Expect(err).To(MatchError(ContainSubstring("can't cd to /tmp/not-a-dir")))
			})
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				_, err := remoteRunner.ChecksumDirectory("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})
})

func destroyInstance(instance *testcluster.Instance) {
	instance.DieInBackground()
	testcluster.WaitForContainersToDie()
}

func makeTmpFile(prefix string) *os.File {
	tmpFile, err := ioutil.TempFile("", prefix)
	Expect(err).NotTo(HaveOccurred())
	return tmpFile
}

func fileSize(file *os.File) int64 {
	archiveFileInfo, err := file.Stat()
	Expect(err).NotTo(HaveOccurred())
	return archiveFileInfo.Size()
}

func resetCursor(file *os.File) *os.File {
	newFile, err := os.Open(file.Name())
	Expect(err).NotTo(HaveOccurred())
	return newFile
}
