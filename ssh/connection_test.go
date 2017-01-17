package ssh_test

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
	"github.com/pivotal-cf/pcf-backup-and-restore/ssh"
	"github.com/pivotal-cf/pcf-backup-and-restore/testcluster"
	"github.com/pivotal-cf/pcf-backup-and-restore/testssh"
	gossh "golang.org/x/crypto/ssh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Connection", func() {
	var conn bosh.SSHConnection
	var connErr error
	var server *testssh.Server
	var privateKey string

	var hostName string

	JustBeforeEach(func() {
		conn, connErr = ssh.ConnectionCreator(hostName, "admin", privateKey)
	})
	AfterEach(func() {
		server.Close()
	})
	BeforeEach(func() {
		privateKey = defaultPrivateKey
		server = testssh.New(GinkgoWriter)
		hostName = "127.0.0.1:" + strconv.Itoa(server.Port)
	})

	Describe("Creates an SSH connection", func() {
		It("does not fail", func() {
			Expect(connErr).NotTo(HaveOccurred())
		})
		It("sshes with the user", func() {
			Expect(server.LastUser).To(Equal("admin"))
		})
		It("uses the given key to connect", func() {
			Expect(server.LastKey).To(Equal(publicKey(privateKey)))
		})
	})

	Describe("fails to connect to the server", func() {
		BeforeEach(func() {
			hostName = "laksdjf"
		})
		It("fails", func() {
			Expect(connErr).To(HaveOccurred())
		})
	})

	Describe("authorization fails", func() {
		BeforeEach(func() {
			server.FailAuth = true
		})
		It("fails", func() {
			Expect(connErr).To(HaveOccurred())
		})
		It("tires to connect with username and key", func() {
			Expect(server.LastUser).To(Equal("admin"))
			Expect(server.LastKey).To(Equal(publicKey(privateKey)))
		})
	})

	Describe("Invalid ssh key", func() {
		BeforeEach(func() {
			privateKey = "laksdjf"
		})
		It("fails", func() {
			Expect(connErr).To(HaveOccurred())
		})
	})

	Describe("username", func() {
		It("returns the SSH username", func() {
			Expect(conn.Username()).To(Equal("admin"))
		})
	})

	Describe("Stream", func() {
		Context("succeeds", func() {
			var writer *bytes.Buffer
			var stdErr []byte
			var exitCode int
			var runError error
			var command string
			JustBeforeEach(func() {
				stdErr, exitCode, runError = conn.Stream(command, writer)
			})
			BeforeEach(func() {
				writer = bytes.NewBufferString("")
				command = makeScript(`#!/usr/bin/env sh
				echo "stdout"
				echo "stderr" >&2`)
			})
			AfterEach(func() {
				if _, err := os.Stat(command); err == nil {
					Expect(os.Remove(command)).To(Succeed())
				}
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
				Expect(exitCode).To(BeZero())
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
			stdOut, stdErr, exitCode, runError = conn.Run(command)
		})
		AfterEach(func() {
			if _, err := os.Stat(command); err == nil {
				Expect(os.Remove(command)).To(Succeed())
			}
		})
		Context("succeds", func() {
			BeforeEach(func() {
				command = makeScript(`#!/usr/bin/env sh
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
		})

		Context("fails to create new session", func() {
			BeforeEach(func() {
				server.FailSession = true
			})
			It("fails", func() {
				Expect(runError).To(HaveOccurred())
			})
		})

		Context("running multiple commands", func() {
			BeforeEach(func() {
				command = "ls"
			})

			It("does not fail", func() {
				_, _, _, runError1 := conn.Run(command)
				_, _, _, runError2 := conn.Run(command)
				_, _, _, runError3 := conn.Run(command)

				Expect(runError1).NotTo(HaveOccurred())
				Expect(runError2).NotTo(HaveOccurred())
				Expect(runError3).NotTo(HaveOccurred())
			})
		})

		Context("exit code not 0", func() {
			BeforeEach(func() {
				command = makeScript(`#!/usr/bin/env sh
				echo "stdout"
				echo "stderr" >&2
				exit 12`)

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
				Expect(exitCode).To(Equal(12))
			})
		})

		Context("command not found", func() {
			BeforeEach(func() {
				command = "foo bar baz"
			})

			It("captures exit code", func() {
				Expect(exitCode).To(Equal(127))
			})
		})
	})
})

var _ = Describe("Connection", func() {
	By("testing against a containerised SSH server", func() {
		Describe("StreamStdin", func() {
			Context("succeeds", func() {
				var reader *bytes.Buffer
				var stdErr []byte
				var stdOut []byte
				var exitCode int
				var runError error
				var connErr error
				var command string
				var conn bosh.SSHConnection
				var instance1 *testcluster.Instance

				JustBeforeEach(func() {
					instance1 = testcluster.NewInstance()
					instance1.CreateUser("test-user", publicKeyForDocker(defaultPrivateKey))

					conn, connErr = ssh.ConnectionCreator(instance1.Address(), "test-user", defaultPrivateKey)
					Expect(connErr).NotTo(HaveOccurred())

					stdOut, stdErr, exitCode, runError = conn.StreamStdin(command, reader)
				})

				BeforeEach(func() {
					reader = bytes.NewBufferString("they will pay for the wall")
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
					Expect(string(stdout)).To(Equal("they will pay for the wall"))
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
		})
	})

})

func publicKey(privateKey string) string {
	parsedPrivateKey, err := gossh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		Fail("Cant parse key")
	}

	return string(parsedPrivateKey.PublicKey().Marshal())
}

func publicKeyForDocker(privateKey string) string {
	parsedPrivateKey, err := gossh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		Fail("Cant parse key")
	}

	return "ssh-rsa " + base64.StdEncoding.EncodeToString(parsedPrivateKey.PublicKey().Marshal())
}

func makeScript(scr string) string {
	file, err := ioutil.TempFile(".", "")
	Expect(err).NotTo(HaveOccurred())
	file.Write([]byte(scr))
	Expect(file.Chmod(0700)).To(Succeed())
	file.Close()
	pwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	return pwd + "/" + file.Name()
}
