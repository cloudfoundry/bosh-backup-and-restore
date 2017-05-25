package ssh_test

import (
	"bytes"
	"encoding/base64"
	"io"
	"log"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
	"github.com/pivotal-cf/bosh-backup-and-restore/testcluster"
	gossh "golang.org/x/crypto/ssh"

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
		instance1 = testcluster.NewInstance()
		instance1.CreateUser("test-user", publicKeyForDocker(defaultPrivateKey))
		privateKey = defaultPrivateKey
		hostname = instance1.Address()
		user = "test-user"

		combinecOutLog := log.New(io.MultiWriter(GinkgoWriter, bytes.NewBufferString("")), "[bosh-package] ", log.Lshortfile)
		combinedErrLog := log.New(io.MultiWriter(GinkgoWriter, bytes.NewBufferString("")), "[bosh-package] ", log.Lshortfile)
		logger = boshlog.New(boshlog.LevelDebug, combinecOutLog, combinedErrLog)
	})

	JustBeforeEach(func() {
		conn, connErr = ssh.NewConnection(hostname, user, privateKey, logger)
	})

	Describe("Connection Creation", func() {
		Describe("Invalid ssh key", func() {
			BeforeEach(func() {
				privateKey = "laksdjf"
			})
			It("fails", func() {
				Expect(connErr).To(HaveOccurred())
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

			AfterEach(func() {
				instance1.DieInBackground()
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
			var writer *bytes.Buffer
			var stdErr []byte
			var exitCode int
			var runError error
			JustBeforeEach(func() {
				Expect(connErr).NotTo(HaveOccurred())
				stdErr, exitCode, runError = conn.Stream("/tmp/foo", writer)
			})
			BeforeEach(func() {
				writer = bytes.NewBufferString("")
				instance1.CreateScript("/tmp/foo", `#!/usr/bin/env sh
				echo "stdout"
				echo "stderr" >&2
				exit 1
				`)
			})
			It("does not fail", func() {
				Expect(runError).NotTo(HaveOccurred())
			})
			It("writes stdout to the writer", func() {
				Expect(writer.String()).To(ContainSubstring("stdout"))
			})
			It("drains stderr", func() {
				Expect(string(stdErr)).To(ContainSubstring("stderr"))
			})
			It("captures exit code", func() {
				Expect(exitCode).To(Equal(1))
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
					Expect(err).To(HaveOccurred())
				})
			})
			Context("Stream", func() {
				JustBeforeEach(func() {
					_, _, err = conn.Stream("ls", bytes.NewBufferString("dont matter"))
				})
				It("fails", func() {
					Expect(err).To(HaveOccurred())
				})
			})
			Context("StreamStdin", func() {
				JustBeforeEach(func() {
					_, _, _, err = conn.StreamStdin("ls", bytes.NewBufferString("dont matter"))
				})
				It("fails", func() {
					Expect(err).To(HaveOccurred())
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
					Expect(err).To(HaveOccurred())
				})
			})
			Context("Stream", func() {
				JustBeforeEach(func() {
					_, _, err = conn.Stream("ls", bytes.NewBufferString("dont matter"))
				})
				It("fails", func() {
					Expect(err).To(HaveOccurred())
				})
			})
			Context("StreamStdin", func() {
				JustBeforeEach(func() {
					_, _, _, err = conn.StreamStdin("ls", bytes.NewBufferString("dont matter"))
				})
				It("fails", func() {
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})

func publicKeyForDocker(privateKey string) string {
	parsedPrivateKey, err := gossh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		Fail("Cant parse key")
	}

	return "ssh-rsa " + base64.StdEncoding.EncodeToString(parsedPrivateKey.PublicKey().Marshal())
}
