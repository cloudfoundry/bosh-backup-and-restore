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
		privateKey = defaultPrivateKey()
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
					instance1.CreateUser("test-user", publicKeyForDocker(defaultPrivateKey()))

					conn, connErr = ssh.ConnectionCreator(instance1.Address(), "test-user", defaultPrivateKey())
					Expect(connErr).NotTo(HaveOccurred())

					stdOut, stdErr, exitCode, runError = conn.StreamStdin(command, reader)
				})

				BeforeEach(func() {
					reader = bytes.NewBufferString("they will pay for the wall")
					command = "cat > /tmp/foo; echo 'here is something on stdout'; echo 'here is something on stderr' >&2"
				})

				AfterEach(func() {
					instance1.Die()
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

func defaultPrivateKey() string {
	return "-----BEGIN RSA PRIVATE KEY-----\nMIIEogIBAAKCAQEA0Pz7vnb5Ieui9oZinuOh2Fyr6wAuWaJMhAClCMrkk0NeLKcX\n2P1hPRvyL9wqXakFlFb1YPIL0pFu0IgIcOGdHF+QTK7o8K4qyeSOK8Qdsi4xokra\nP89FLcT93NbsE3grqXSE7ENKJHGP68pEwc0gTLD1FZ9O7x1IhihuDjtUflO/juAU\n7b5gycg9cJUtpM97k3i/u0dKvy3qMm14mLx4SRhRlVaBTzmGIuUqtOf3aJj4M8SU\ni3j77RGakCEwMCMAZB9i3eYN1L1Ft66DPjDphsp1Hp1NTPxvhCUZyTsPxboSiCDS\nzfZvTLygeZ88dvB3vldouaR0KHk9lbu1HzrKuQIDAQABAoIBAF7FCgfh/bHDIEA4\nyooQ4biytYc4qswczCPkAvLMxwB8wTzwfOD6bdj/TkEjztZwKkaNdHKE8JWJO742\nodVGii9uqooLmzhhUqgBC/OO2ISPbBSTawsam91YgmJd1+owSWRroUdecEW8da5Q\nKAPWWDpO2KT4fBv0pImp1daAUx2BXJR4PR5WbdA9ql6t/oT1ptLEI6KOixkbKZ/Z\nn1RrSnweuoJQira3IcuUlO/4XD//yuvuFj4rYINUbwpH/auMd9fdQXwZKInS4IBV\nwMajoSWB1hX4+YzLzc+V6R7ub35+RFW2X8g/QUbfLixpgx/SM0MgCd0GFd4Oa21C\nrEpRWAECgYEA9xrKZzG6twO6SkTSgRViwOUyXKCNWygMHYv6gF6rsJ9KOY/7OYv2\nK0MSv9RnWp9Js2fAGH80YZTMOCG2qTd85dOEd7z/DldlX2DkwurL2yyZEOdZGv9W\n89RQbx5ScGWmuuIePuE9BXS0LxGvD+Yt9AtN/smoQeNEdoEmlAnlOwECgYEA2ILt\nnk90q09OXm4B+g1nPD/M7Q99+giX+AWLtSHCkvfClIBxG6Xs0qD88MBTk5MrkHcB\ndaU/YrNmMS3GuTZS8wPZ6xu2DmjE/lX55xXg6QbPNCslXMfvw+URWKnckWDNGut5\n3EkkzMZYLrytTrVkd0Q7aOCcDDH1wNrvyXqWJ7kCgYASfI+d7suAO6gpPELfY2Ey\n+zKsWVqZ8kINx9Yi2nJP0Wr1KX9rC7yL+gWiElr1Haue32kwq/uYPVCV9ne66yrN\n6ugjKSGPyhwMaaxTpMtBh3GgIR66dVXlAgJOfd8/B2vU2WvX2nP9P4DncJQ/RUI0\n2s+n+yA6Za1OjFT9iEv9AQKBgDjx7rdlpITuHemeO2zeG5nwGeD74yFhIz87jiw8\nzeVDvuy5/4XLFUesyfo0S4cT/TBI7JxZsxstniIvLQZHsHd0OtuodTDDA5T1Xf4W\ndgo0HUlWU8RcXcaDOBW+z2F5OVjsOCflIQWu4UChpV9/PAZWbt29va1DcqSfsNOo\nJ1gZAoGAF3x4r4LjeFJVhdAAdxSG6LpSUj3ZsBVn+YCrYgZsNMlmCD6+xnbtw9rZ\nL5UoyIJeGTb89xnWaYqXnMqkGF8c4coKb2CG05a6yC1EwL4DSbCDNvVhniARIStL\ng/e1bAStuIe3sA6ixeA/Z0FzkzqBx7yY9oEtDdvPaUA07IeriQg=\n-----END RSA PRIVATE KEY-----\n"
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
