package deployment

import (
	"io/ioutil"
	"os"

	"time"

	"fmt"

	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

var _ = Describe("CLI Interface", func() {

	var director *mockhttp.Server
	var backupWorkspace string

	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
		director.VerifyMocks()
	})

	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	Context("bbr deployment", func() {
		AssertDeploymentCLIBehaviour := func(cmd string) {
			Context("params", func() {
				It("can invoke command with short names", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{},
						"deployment",
						"--ca-cert", sslCertPath,
						"-u", "admin",
						"-p", "admin",
						"-t", director.URL,
						"-d", "my-new-deployment",
						cmd)

					director.VerifyMocks()
				})

				It("can invoke command with long names", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{},
						"deployment",
						"--ca-cert", sslCertPath,
						"--username", "admin",
						"--password", "admin",
						"--target", director.URL,
						"--deployment", "my-new-deployment",
						cmd)

					director.VerifyMocks()
				})
			})

			Context("password is supported from env", func() {
				It("can invoke command with long names", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd)

					director.VerifyMocks()
				})
			})

			Context("Hostname is malformed", func() {
				var output helpText
				var session *gexec.Session
				BeforeEach(func() {
					badDirectorURL := "https://:25555"
					session = binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--username", "admin", "--password", "admin", "--target", badDirectorURL, "--deployment", "my-new-deployment", cmd)
					output.output = session.Err.Contents()
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(output.outputString()).To(ContainSubstring("invalid bosh URL"))
				})
			})

			Context("Custom CA cert cannot be read", func() {
				var output helpText
				var session *gexec.Session
				BeforeEach(func() {
					session = binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--ca-cert", "/tmp/whatever", "--username", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd)
					output.output = session.Err.Contents()
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(output.outputString()).To(ContainSubstring("open /tmp/whatever: no such file or directory"))
				})
			})

			Context("Wrong global args", func() {
				var output helpText
				var session *gexec.Session
				BeforeEach(func() {
					session = binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--dave", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd)
					output.output = session.Out.Contents()
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(output.outputString()).To(ContainSubstring("Incorrect Usage"))
				})

				It("displays the usable flags", func() {
					ShowsTheDeploymentHelpText(&output)
				})
			})

			Context("when any required flags are missing", func() {
				var output helpText
				var session *gexec.Session
				var command []string
				var env []string
				BeforeEach(func() {
					env = []string{"BOSH_CLIENT_SECRET=admin"}
				})
				JustBeforeEach(func() {
					session = binary.Run(backupWorkspace, env, command...)
					output.output = session.Out.Contents()
				})

				Context("Missing target", func() {
					BeforeEach(func() {
						command = []string{"deployment", "--username", "admin", "--password", "admin", "--deployment", "my-new-deployment", cmd}
					})
					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("--target flag is required."))
					})

					It("displays the usable flags", func() {
						ShowsTheDeploymentHelpText(&output)
					})
				})

				Context("Missing username", func() {
					BeforeEach(func() {
						command = []string{"deployment", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd}
					})
					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("--username flag is required."))
					})

					It("displays the usable flags", func() {
						ShowsTheDeploymentHelpText(&output)
					})
				})

				Context("Missing password in args", func() {
					BeforeEach(func() {
						env = []string{}
						command = []string{"deployment", "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd}
					})
					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("--password flag is required."))
					})

					It("displays the usable flags", func() {
						ShowsTheDeploymentHelpText(&output)
					})
				})

				Context("Missing deployment", func() {
					BeforeEach(func() {
						command = []string{"deployment", "--username", "admin", "--password", "admin", "--target", director.URL, cmd}
					})
					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("--deployment flag is required."))
					})

					It("displays the usable flags", func() {
						ShowsTheDeploymentHelpText(&output)
					})
				})
			})

			Context("with debug flag set", func() {
				It("outputs verbose HTTP logs", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					session := binary.Run(backupWorkspace, []string{}, "deployment", "--debug", "--ca-cert", sslCertPath, "--username", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd)

					Expect(string(session.Out.Contents())).To(ContainSubstring("Sending GET request to endpoint"))

					director.VerifyMocks()
				})
			})
		}

		Context("backup", func() {
			AssertDeploymentCLIBehaviour("backup")
		})

		Context("restore", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(backupWorkspace+"/"+"my-new-deployment", 0777)).To(Succeed())
				createFileWithContents(backupWorkspace+"/"+"my-new-deployment"+"/"+"metadata", []byte(`---
instances: []`))
			})

			AssertDeploymentCLIBehaviour("restore")
		})

		Context("--help", func() {
			var output helpText

			BeforeEach(func() {
				output.output = binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--help").Out.Contents()
			})

			It("displays the usable flags", func() {
				ShowsTheDeploymentHelpText(&output)
			})
		})

		Context("no arguments", func() {
			var output helpText

			BeforeEach(func() {
				output.output = binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment").Out.Contents()
			})

			It("displays the usable flags", func() {
				ShowsTheDeploymentHelpText(&output)
			})
		})

	})

	Context("bbr director", func() {
		Describe("invalid command line arguments", func() {
			Context("private-key-path flag", func() {
				var (
					keyFile       *os.File
					err           error
					session       *gexec.Session
					sessionOutput string
				)

				BeforeEach(func() {
					keyFile, err = ioutil.TempFile("", time.Now().String())
					Expect(err).NotTo(HaveOccurred())
					fmt.Fprintf(keyFile, "this is not a valid key")

					session = binary.Run(backupWorkspace,
						[]string{},
						"director",
						"--artifactname", "foo",
						"-u", "admin",
						"--host", "10.0.0.5",
						"--private-key-path", keyFile.Name(),
						"backup")
					Eventually(session).Should(gexec.Exit())
					sessionOutput = string(session.Err.Contents())
				})

				It("prints a meaningful message when the key is invalid", func() {
					Expect(sessionOutput).To(ContainSubstring("ssh.NewConnection.ParsePrivateKey failed"))
				})

				It("doesn't print a stack trace", func() {
					Expect(sessionOutput).NotTo(ContainSubstring("main.go"))
				})

				It("saves the stack trace into a file", func() {
					files, err := filepath.Glob(filepath.Join(backupWorkspace, "bbr-*.err.log"))
					Expect(err).NotTo(HaveOccurred())
					logFilePath := files[0]
					_, err = os.Stat(logFilePath)
					Expect(os.IsNotExist(err)).To(BeFalse())
					stackTrace, err := ioutil.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})
			})
		})
	})

	Context("bbr with no arguments", func() {
		var output helpText

		BeforeEach(func() {
			output.output = binary.Run(backupWorkspace, []string{""}).Out.Contents()
		})

		It("displays the usable flags", func() {
			ShowsTheMainHelpText(&output)
		})
	})

})
