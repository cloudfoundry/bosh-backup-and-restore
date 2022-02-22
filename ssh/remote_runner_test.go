package ssh_test

import (
	"bytes"
	"io"
	"log"

	"io/ioutil"

	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gossh "golang.org/x/crypto/ssh"
)

var userPrivateKey string
var user string
var testInstance *testcluster.Instance
var sshConnection ssh.SSHConnection

var _ = Describe("SshRemoteRunner", func() {
	var sshRemoteRunner ssh.RemoteRunner

	BeforeEach(func() {
		user = "test-user"
		userPrivateKey = defaultPrivateKey

		testInstance = testcluster.NewInstanceWithKeepAlive(2)
		testInstance.CreateUser(user, publicKeyForDocker(userPrivateKey))

		hostPublicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(testInstance.HostPublicKey()))
		Expect(err).NotTo(HaveOccurred())

		combinedLog := log.New(io.MultiWriter(GinkgoWriter, bytes.NewBufferString("")), "[bosh-package] ", log.Lshortfile)
		logger := boshlog.New(boshlog.LevelDebug, combinedLog)

		sshConnection, err = ssh.NewConnection(testInstance.Address(), user, userPrivateKey, gossh.FixedHostKey(hostPublicKey),
			[]string{hostPublicKey.Type()}, logger)
		Expect(err).NotTo(HaveOccurred())

		sshRemoteRunner, err = ssh.NewSshRemoteRunner(testInstance.Address(), user, userPrivateKey, gossh.FixedHostKey(hostPublicKey),
			[]string{hostPublicKey.Type()}, logger)
		Expect(err).NotTo(HaveOccurred())
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

	var makeAccessibleOnlyByRoot = func(path string) {
		runCommand("sudo chown root:root " + path)
		runCommand("sudo chmod 0700 " + path)
	}
	Describe("ConnectedUsername", func() {
		It("returns the name of the connected user", func() {
			Expect(sshRemoteRunner.ConnectedUsername()).To(Equal(user))
		})
	})

	Describe("DirectoryExists", func() {
		Context("When the directory does not exist", func() {
			It("returns false", func() {
				Expect(sshRemoteRunner.DirectoryExists("/tmp/non-existing-dir")).To(BeFalse())
			})
		})

		Context("When the directory exists", func() {
			BeforeEach(func() {
				runCommand("mkdir -p /tmp/an-existing-dir")
				makeAccessibleOnlyByRoot("/tmp")
			})

			It("returns false", func() {
				Expect(sshRemoteRunner.DirectoryExists("/tmp/an-existing-dir")).To(BeTrue())
			})
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				_, err := sshRemoteRunner.DirectoryExists("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("CreateDirectory", func() {
		It("creates a directory", func() {
			makeAccessibleOnlyByRoot("/tmp")
			Expect(sshRemoteRunner.CreateDirectory("/tmp/a-new-directory")).To(Succeed())
			Expect(sshRemoteRunner.DirectoryExists("/tmp/a-new-directory")).To(BeTrue())
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				err := sshRemoteRunner.CreateDirectory("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("RemoveDirectory", func() {
		Context("When the directory exists", func() {
			BeforeEach(func() {
				runCommand("mkdir -p /tmp/existing-directory")
				makeAccessibleOnlyByRoot("/tmp/existing-directory")
			})

			It("removes the directory", func() {
				err := sshRemoteRunner.RemoveDirectory("/tmp/existing-directory")

				Expect(err).NotTo(HaveOccurred())
				Expect(sshRemoteRunner.DirectoryExists("/tmp/non-existing-dir")).To(BeFalse())
			})
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				err := sshRemoteRunner.RemoveDirectory("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("remote archiving and extracting of directories", func() {
		It("archives and extracts directories from/to remote servers", func() {
			runCommand("mkdir -p /tmp/dir-to-archive")
			runCommand("echo 'one' > /tmp/dir-to-archive/file1")
			runCommand("echo 'two' > /tmp/dir-to-archive/file2")
			makeAccessibleOnlyByRoot("/tmp/dir-to-archive")

			By("downloading and archiving the directory")
			archiveFile := makeTmpFile("remote-runner-test-")
			err := sshRemoteRunner.ArchiveAndDownload("/tmp/dir-to-archive", archiveFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(fileSize(archiveFile)).To(BeNumerically(">", 0))

			By("by extracting and uploading the archive to a specified directory")
			runCommand("mkdir -p /tmp/uploaded-dir")
			makeAccessibleOnlyByRoot("/tmp/uploaded-dir")
			archiveFile = resetCursor(archiveFile)
			err = sshRemoteRunner.ExtractAndUpload(archiveFile, "/tmp/uploaded-dir")
			Expect(err).NotTo(HaveOccurred())
			lsOutput := runCommand("sudo ls /tmp/uploaded-dir")
			Expect(lsOutput).To(Equal("file1\nfile2\n"))
		})

		Context("when archiving fails", func() {
			Context("when the command fails", func() {
				It("returns an error", func() {
					archiveFile := makeTmpFile("remote-runner-test-")
					err := sshRemoteRunner.ArchiveAndDownload("/tmp/unexisting-dir", archiveFile)
					Expect(err).To(MatchError(ContainSubstring("No such file or directory")))
				})
			})

			Context("when the connection fails", func() {
				BeforeEach(func() {
					destroyInstance(testInstance)
				})

				It("returns an error", func() {
					archiveFile := makeTmpFile("remote-runner-test-")
					err := sshRemoteRunner.ArchiveAndDownload("/tmp/unexisting-dir", archiveFile)
					Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
				})
			})
		})

		Context("when extracting fails", func() {
			Context("when the command fails", func() {
				It("returns an error", func() {
					notATar := makeTmpFile("remote-runner-test-")
					err := sshRemoteRunner.ExtractAndUpload(notATar, "/tmp/arbitrary-dir")
					Expect(err).To(MatchError(ContainSubstring("This does not look like a tar archive")))
				})
			})

			Context("when the connection fails", func() {
				BeforeEach(func() {
					destroyInstance(testInstance)
				})

				It("returns an error", func() {
					arbitraryFile := makeTmpFile("remote-runner-test-")
					err := sshRemoteRunner.ExtractAndUpload(arbitraryFile, "/tmp/arbitrary-dir")
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
				makeAccessibleOnlyByRoot("/tmp/a-dir")
			})

			It("returns a string with the specified file or directory size", func() {
				Expect(sshRemoteRunner.SizeOf("/tmp/a-dir")).To(Equal("1.5M"))
			})
		})

		Context("when the directory does not exist", func() {
			It("returns an error", func() {
				_, err := sshRemoteRunner.SizeOf("/tmp/not-a-file")
				Expect(err).To(MatchError(ContainSubstring("No such file or directory")))
			})
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				_, err := sshRemoteRunner.SizeOf("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("SizeInBytes", func() {
		Context("when the file or directory exists", func() {
			BeforeEach(func() {
				runCommand("mkdir /tmp/a-dir")
				runCommand("dd if=/dev/zero of=/tmp/a-dir/a-file bs=1k count=1000")
				runCommand("dd if=/dev/zero of=/tmp/a-dir/b-file bs=1k count=500")
				makeAccessibleOnlyByRoot("/tmp/a-dir")
			})

			It("returns a string with the specified file or directory size", func() {
				Expect(sshRemoteRunner.SizeInBytes("/tmp/a-dir")).To(Equal(1540096))
			})
		})

		Context("when the directory does not exist", func() {
			It("returns an error", func() {
				_, err := sshRemoteRunner.SizeInBytes("/tmp/not-a-file")
				Expect(err).To(MatchError(ContainSubstring("No such file or directory")))
			})
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				_, err := sshRemoteRunner.SizeInBytes("whatever")
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
				makeAccessibleOnlyByRoot("/tmp/a-dir")
			})

			It("calculates the SHA256 checksum for each file in the directory", func() {
				Expect(sshRemoteRunner.ChecksumDirectory("/tmp/a-dir")).To(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("./file1", "b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c"),
					HaveKeyWithValue("./file2", "7d865e959b2466918c9863afca942d0fb89d7c9ac0c99bafc3749504ded97730"),
				))
			})
		})

		Context("when the directory does not exist", func() {
			It("returns an error", func() {
				_, err := sshRemoteRunner.ChecksumDirectory("/tmp/not-a-dir")
				Expect(err).To(MatchError(ContainSubstring("can't cd to /tmp/not-a-dir")))
			})
		})

		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				_, err := sshRemoteRunner.ChecksumDirectory("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("RunScriptWithEnv", func() {
		Context("When the script exists", func() {
			It("runs the script with the specified env variables", func() {

				runCommand("echo 'env' > /tmp/example-script")
				makeAccessibleOnlyByRoot("/tmp/example-script")

				stdoutBuffer := &bytes.Buffer{}

				err := sshRemoteRunner.RunScriptWithEnv("/tmp/example-script", map[string]string{"env1": "foo", "env2": "bar"}, "", stdoutBuffer)

				Expect(err).NotTo(HaveOccurred())

				Expect(stdoutBuffer.Bytes()).To(SatisfyAll(
					ContainSubstring("env1=foo"),
					ContainSubstring("env2=bar"),
				))
			})
		})

		Context("when the script is not there", func() {
			It("returns a helpful error", func() {
				err := sshRemoteRunner.RunScriptWithEnv("/tmp/example-script", map[string]string{"env1": "foo", "env2": "bar"}, "", io.Discard)

				Expect(err).To(MatchError(ContainSubstring("command not found")))

			})
		})

		Context("When the script errors", func() {
			It("returns an error containing the script stderr", func() {
				runCommand("echo '>&2 echo example script has errorred; exit 12' > /tmp/example-script")
				runCommand("chmod +x /tmp/example-script")

				err := sshRemoteRunner.RunScriptWithEnv("/tmp/example-script", map[string]string{"env1": "foo", "env2": "bar"}, "", io.Discard)

				Expect(err).To(MatchError(ContainSubstring("example script has errorred - exit code 12")))

			})
		})
		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				err := sshRemoteRunner.RunScriptWithEnv("whatever", map[string]string{}, "", io.Discard)
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})

	Describe("FindFiles", func() {
		Context("when there are files that match the pattern", func() {
			It("returns exactly those files", func() {
				runCommand("touch /tmp/script-to-find")
				runCommand("touch /tmp/script-to-find2")
				runCommand("touch /tmp/script-to-not-find")
				makeAccessibleOnlyByRoot("/tmp")

				files, err := sshRemoteRunner.FindFiles("/tmp/*to-find*")
				Expect(err).NotTo(HaveOccurred())
				Expect(files).To(ConsistOf(
					"/tmp/script-to-find",
					"/tmp/script-to-find2",
				))

			})
		})

		Context("when there are no files that match the pattern", func() {
			It("returns exactly those files", func() {
				files, err := sshRemoteRunner.FindFiles("/tmp/this-file")
				Expect(err).NotTo(HaveOccurred())
				Expect(files).To(HaveLen(0))
			})
		})

		Context("when the find command errors", func() {
			It("bubbles the error up", func() {
				_, err := sshRemoteRunner.FindFiles("; cause-an-error")
				Expect(err).To(MatchError(ContainSubstring("not found")))
			})
		})
		Context("When the ssh connection fails", func() {
			BeforeEach(func() {
				destroyInstance(testInstance)
			})

			It("returns an error", func() {
				_, err := sshRemoteRunner.FindFiles("whatever")
				Expect(err).To(MatchError(ContainSubstring("ssh.Dial failed")))
			})
		})
	})
})

func destroyInstance(ssh *testcluster.Instance) {
	ssh.DieInBackground()
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
