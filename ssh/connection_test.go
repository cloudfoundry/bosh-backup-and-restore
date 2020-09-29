package ssh_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"log"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	gossh "golang.org/x/crypto/ssh"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Connection", func() {
	var conn ssh.SSHConnection
	var connErr error
	var hostname string
	var user string
	var privateKey string

	var instance1 *testcluster.Instance
	var logger ssh.Logger

	BeforeEach(func() {
		instance1 = testcluster.NewInstanceWithKeepAlive(2)
		instance1.CreateUser("test-user", publicKeyForDocker(defaultPrivateKey))
		privateKey = defaultPrivateKey
		hostname = instance1.Address()
		user = "test-user"

		combinedOutLog := log.New(io.MultiWriter(GinkgoWriter, bytes.NewBufferString("")), "[bosh-package] ", log.Lshortfile)
		logger = boshlog.New(boshlog.LevelDebug, combinedOutLog)
		ssh.ResetBuildSSHSession()
	})

	AfterEach(func() {
		instance1.DieInBackground()
	})

	JustBeforeEach(func() {
		hostPublicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(instance1.HostPublicKey()))
		Expect(err).NotTo(HaveOccurred())

		conn, connErr = ssh.NewConnection(hostname, user, privateKey, gossh.FixedHostKey(hostPublicKey), []string{hostPublicKey.Type()}, logger)
	})

	Describe("Connection Creation", func() {
		Describe("Invalid ssh key", func() {
			BeforeEach(func() {
				privateKey = "laksdjf"
			})
			It("fails", func() {
				Expect(connErr).To(MatchError(ContainSubstring("ssh.NewConnection.ParsePrivateKey failed")))
			})
		})
	})

	Describe("Username", func() {
		It("returns the SSH username", func() {
			Expect(conn.Username()).To(Equal("test-user"))
		})
	})

	Context("successful connections", func() {
		Describe("StreamStdin", func() {
			var reader *bytes.Buffer
			var stdErr []byte
			var stdOut []byte
			var exitCode int
			var runError error
			var command string

			JustBeforeEach(func() {
				Expect(connErr).NotTo(HaveOccurred())
				stdOut, stdErr, exitCode, runError = conn.StreamStdin(command, reader)
			})

			BeforeEach(func() {
				reader = bytes.NewBufferString("I am from the reader")
				command = "cat > /tmp/foo; echo 'here is something on stdout'; echo 'here is something on stderr' >&2"
			})

			It("does not fail", func() {
				Expect(runError).NotTo(HaveOccurred())
			})

			It("reads stdout from the reader", func() {
				stdout, _, _, _ := conn.Run("cat /tmp/foo")
				Expect(string(stdout)).To(Equal("I am from the reader"))
			})

			It("drains stdout", func() {
				Expect(string(stdOut)).To(ContainSubstring("here is something on stdout"))
			})

			It("drains stderr", func() {
				Expect(string(stdErr)).To(ContainSubstring("here is something on stderr"))
			})
			It("captures exit code", func() {
				Expect(exitCode).To(BeZero())
			})
		})

		Describe("Stream", func() {
			var stdout io.Writer
			var stdErr []byte
			var exitCode int
			var runError error
			var command string
			JustBeforeEach(func() {
				Expect(connErr).NotTo(HaveOccurred())
				stdErr, exitCode, runError = conn.Stream(command, stdout)
			})
			Context("success", func() {
				BeforeEach(func() {
					command = "/tmp/foo"
					stdout = bytes.NewBufferString("")
					instance1.CreateScript("/tmp/foo", `#!/usr/bin/env sh
				echo "stdout"
				echo "stderr" >&2
				exit 1`)
				})
				It("does not fail", func() {
					Expect(runError).NotTo(HaveOccurred())
				})
				It("writes stdout to the writer", func() {
					Expect(stdout.(*bytes.Buffer).String()).To(ContainSubstring("stdout"))
				})
				It("drains stderr", func() {
					Expect(string(stdErr)).To(ContainSubstring("stderr"))
				})
				It("captures exit code", func() {
					Expect(exitCode).To(Equal(1))
				})
			})
		})

		Describe("Run", func() {
			var stdOut []byte
			var stdErr []byte
			var exitCode int
			var runError error
			var command string
			JustBeforeEach(func() {
				Expect(connErr).NotTo(HaveOccurred())
				stdOut, stdErr, exitCode, runError = conn.Run(command)
			})
			BeforeEach(func() {
				command = "/tmp/foo"
				instance1.CreateScript(command, `#!/usr/bin/env sh
				echo "stdout"
				echo "stderr" >&2`)

			})
			It("does not fail", func() {
				Expect(runError).NotTo(HaveOccurred())
			})
			It("drains stdout", func() {
				Expect(string(stdOut)).To(ContainSubstring("stdout"))
			})
			It("drains stderr", func() {
				Expect(string(stdErr)).To(ContainSubstring("stderr"))
			})
			It("captures exit code", func() {
				Expect(exitCode).To(BeZero())
			})
			It("closes the connection after executing the command", func() {
				Eventually(instance1.Run("ps", "auxwww")).ShouldNot(ContainSubstring(user))
			})
			Context("running multiple commands", func() {

				It("does not fail", func() {
					_, _, _, runError1 := conn.Run("ls")
					_, _, _, runError2 := conn.Run("ls")
					_, _, _, runError3 := conn.Run("ls")

					Expect(runError1).NotTo(HaveOccurred())
					Expect(runError2).NotTo(HaveOccurred())
					Expect(runError3).NotTo(HaveOccurred())
				})
			})

			Context("exit code not 0", func() {
				BeforeEach(func() {
					command = "/tmp/foo"
					instance1.CreateScript(command, `#!/usr/bin/env sh
				exit 12`)

				})
				It("does not fail", func() {
					Expect(runError).NotTo(HaveOccurred())
				})
				It("captures exit code", func() {
					Expect(exitCode).To(Equal(12))
				})
			})

			Context("command not found", func() {
				BeforeEach(func() {
					command = "/tmp/non-existent"
				})
				It("captures exit code", func() {
					Expect(exitCode).To(Equal(127))
				})
			})

			When("the network dies before the command finishes", func() {
				var (
					fakeSSHSession *fakes.FakeSSHSession
					fakeLogger     *fakes.FakeLogger
				)

				BeforeEach(func() {
					fakeLogger = new(fakes.FakeLogger)
					logger = fakeLogger
					fakeSSHSession = new(fakes.FakeSSHSession)
					ssh.InjectBuildSSHSession(func(client *gossh.Client, stdin io.Reader, stdout, stderr io.Writer) (ssh.SSHSession, error) {
						return fakeSSHSession, nil
					})

					fakeSSHSession.RunReturns(new(gossh.ExitMissingError))
				})

				It("returns a helpful error message", func() {
					Expect(runError).To(MatchError(ContainSubstring("ssh session ended before returning an exit code")))
					Expect(fakeLogger.ErrorCallCount()).To(Equal(1))
					tag, msg, _ := fakeLogger.ErrorArgsForCall(0)
					Expect(tag).To(Equal("bbr"))
					Expect(msg).To(ContainSubstring("Did the network just fail? It looks like my ssh session to %s ended suddenly without getting an exit status from the remote VM", hostname))
				})
			})
		})
	})

	Context("connection failures", func() {
		Describe("unreachable host", func() {
			var err error

			BeforeEach(func() {
				hostname = "laksdjf"
			})

			Context("Run", func() {
				JustBeforeEach(func() {
					_, _, _, err = conn.Run("ls")
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("ssh.Run failed")))
				})
			})

			Context("Stream", func() {
				JustBeforeEach(func() {
					_, _, err = conn.Stream("ls", bytes.NewBufferString("dont matter"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("ssh.Stream failed")))
				})
			})

			Context("StreamStdin", func() {
				JustBeforeEach(func() {
					_, _, _, err = conn.StreamStdin("ls", bytes.NewBufferString("dont matter"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("ssh.StreamStdin failed")))
				})
			})
		})

		Describe("authorization failure", func() {
			var err error

			BeforeEach(func() {
				user = "foo"
			})

			Context("Run", func() {
				JustBeforeEach(func() {
					_, _, _, err = conn.Run("ls")
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("ssh.Run failed")))
				})
			})

			Context("Stream", func() {
				JustBeforeEach(func() {
					_, _, err = conn.Stream("ls", bytes.NewBufferString("dont matter"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("ssh.Stream failed")))
				})
			})

			Context("StreamStdin", func() {
				JustBeforeEach(func() {
					_, _, _, err = conn.StreamStdin("ls", bytes.NewBufferString("dont matter"))
				})

				It("fails", func() {
					Expect(err).To(MatchError(ContainSubstring("ssh.StreamStdin failed")))
				})
			})
		})
	})

	When("working with a long running remote process", func() {
		var stdOut []byte

		BeforeEach(func() {
			instance1.CreateScript("/tmp/produce", `#!/usr/bin/env sh
				echo "start"
				sleep 4
				echo "end"`)

			conn, connErr = ssh.NewConnectionWithServerAliveInterval(hostname, user, privateKey, gossh.InsecureIgnoreHostKey(), nil, 1, logger)
			Expect(connErr).NotTo(HaveOccurred())

			stdOut, _, _, _ = conn.Run("/tmp/produce")
		})

		It("keeps the connection alive", func() {
			Eventually(stdOut).Should(ContainSubstring("start"))
			Eventually(stdOut).Should(ContainSubstring("end"))
		})
	})

	Context("when streaming stdout from the server fails locally", func() {
		var stdout io.Writer
		var stdErr []byte
		var runError error
		var command string

		BeforeEach(func() {
			command = "echo 'about to sleep' && sleep 5 && >&2 echo 'error'"

			stdout = errorWriter{errorMessage: "I am error"}

			rapidKeepAliveSignalInterval := time.Duration(1)
			conn, connErr = ssh.NewConnectionWithServerAliveInterval(
				hostname,
				user,
				privateKey,
				gossh.InsecureIgnoreHostKey(),
				nil,
				rapidKeepAliveSignalInterval,
				logger)
			Expect(connErr).NotTo(HaveOccurred())
			stdErr, _, runError = conn.Stream(command, stdout)
		})

		It("does not hang forever", func() {
			By("not continuing to run the command after receiving an error from the stdout writer", func() {
				Expect(string(stdErr)).NotTo(ContainSubstring("error"))
			})

			By("returning the error", func() {
				Expect(runError).To(MatchError(ContainSubstring("I am error")))
			})

			By("closing the ssh connection", func() {
				Eventually(instance1.Run("ps", "auxwww")).ShouldNot(ContainSubstring(user))
			})
		})
	})
})

type errorWriter struct {
	errorMessage string
}

func (ew errorWriter) Write([]byte) (int, error) {
	return 0, errors.New(ew.errorMessage)
}

func publicKeyForDocker(privateKey string) string {
	parsedPrivateKey, err := gossh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		Fail("Cant parse key")
	}

	return "ssh-rsa " + base64.StdEncoding.EncodeToString(parsedPrivateKey.PublicKey().Marshal())
}
