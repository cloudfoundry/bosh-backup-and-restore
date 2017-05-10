package integration

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

	AssertCLIBehaviour := func(cmd string) {
		Context("params", func() {
			It("can invoke command with short names", func() {
				director.VerifyAndMock(
					mockbosh.Info().WithAuthTypeBasic(),
					mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
				)

				runBinary(backupWorkspace,
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

				runBinary(backupWorkspace,
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

				runBinary(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd)

				director.VerifyMocks()
			})
		})

		Context("Hostname is malformed", func() {
			var output helpText
			var session *gexec.Session
			BeforeEach(func() {
				badDirectorURL := "https://:25555"
				session = runBinary(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--username", "admin", "--password", "admin", "--target", badDirectorURL, "--deployment", "my-new-deployment", cmd)
				output.output = session.Err.Contents()
			})

			It("Exits with non zero", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("displays a failure message", func() {
				Expect(output.outputString()).To(ContainSubstring("Target director URL is malformed"))
			})
		})

		Context("Custom CA cert cannot be read", func() {
			var output helpText
			var session *gexec.Session
			BeforeEach(func() {
				session = runBinary(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--ca-cert", "/tmp/whatever", "--username", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd)
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
				session = runBinary(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--dave", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd)
				output.output = session.Out.Contents()
			})

			It("Exits with non zero", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("displays a failure message", func() {
				Expect(output.outputString()).To(ContainSubstring("Incorrect Usage"))
			})
			ShowsTheBackupHelpText(&output)
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
				session = runBinary(backupWorkspace, env, command...)
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
				ShowsTheBackupHelpText(&output)
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
				ShowsTheBackupHelpText(&output)
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
				ShowsTheBackupHelpText(&output)
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
				ShowsTheBackupHelpText(&output)
			})
		})
		Context("with debug flag set", func() {
			It("outputs verbose HTTP logs", func() {
				director.VerifyAndMock(
					mockbosh.Info().WithAuthTypeBasic(),
					mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
				)

				session := runBinary(backupWorkspace, []string{}, "deployment", "--debug", "--ca-cert", sslCertPath, "--username", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", cmd)

				Expect(string(session.Out.Contents())).To(ContainSubstring("Sending GET request to endpoint"))

				director.VerifyMocks()
			})
		})
	}
	Context("backup", func() {
		AssertCLIBehaviour("backup")
	})

	Context("restore", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(backupWorkspace+"/"+"my-new-deployment", 0777)).To(Succeed())
			createFileWithContents(backupWorkspace+"/"+"my-new-deployment"+"/"+"metadata", []byte(`---
instances: []`))

		})
		AssertCLIBehaviour("restore")
	})

	Context("Help", func() {
		var output helpText

		BeforeEach(func() {
			output.output = runBinary(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--help").Out.Contents()
		})

		ShowsTheBackupHelpText(&output)
	})

	Context("deployment - no arguments", func() {
		var output helpText

		BeforeEach(func() {
			output.output = runBinary(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "").Out.Contents()
		})

		ShowsTheBackupHelpText(&output)
	})
})
