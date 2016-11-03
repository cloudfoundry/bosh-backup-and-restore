package integration

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

var _ = Describe("CLI Interface", func() {

	var director *mockhttp.Server
	AfterEach(func() {
		director.VerifyMocks()
	})
	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
	})

	Context("Params", func() {
		It("can invoke command with short names", func() {
			director.VerifyAndMock(mockbosh.VMsForDeployment("my-new-deployment").NotFound())

			runBinary([]string{}, "--ca-cert", sslCertPath, "-u", "admin", "-p", "admin", "-t", director.URL, "-d", "my-new-deployment", "backup")

			director.VerifyMocks()
		})
		It("can invoke command with long names", func() {
			director.VerifyAndMock(mockbosh.VMsForDeployment("my-new-deployment").NotFound())

			runBinary([]string{}, "--ca-cert", sslCertPath, "--username", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")

			director.VerifyMocks()
		})
	})

	Context("with debug flag set", func() {
		It("outputs verbose HTTP logs", func() {
			director.VerifyAndMock(mockbosh.VMsForDeployment("my-new-deployment").NotFound())

			session := runBinary([]string{}, "--debug", "--ca-cert", sslCertPath, "--username", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")

			Expect(string(session.Out.Contents())).To(ContainSubstring("Sending GET request to endpoint"))

			director.VerifyMocks()
		})
	})

	Context("password is supported from env", func() {
		It("can invoke command with long names", func() {
			director.VerifyAndMock(mockbosh.VMsForDeployment("my-new-deployment").NotFound())

			runBinary([]string{"BOSH_PASSWORD=admin"}, "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")

			director.VerifyMocks()
		})
	})

	Context("Hostname is malformed", func() {
		var output helpText
		var session *gexec.Session
		BeforeEach(func() {
			badDirectorURL := "https://:25555"
			session = runBinary([]string{"BOSH_PASSWORD=admin"}, "--username", "admin", "--password", "admin", "--target", badDirectorURL, "--deployment", "my-new-deployment", "backup")
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
			session = runBinary([]string{"BOSH_PASSWORD=admin"}, "--ca-cert", "/tmp/whatever", "--username", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")
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
			session = runBinary([]string{"BOSH_PASSWORD=admin"}, "--dave", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")
			output.output = session.Out.Contents()
		})

		It("Exits with non zero", func() {
			Expect(session.ExitCode()).NotTo(BeZero())
		})

		It("displays a failure message", func() {
			Expect(output.outputString()).To(ContainSubstring("Incorrect Usage"))
		})
		ShowsTheHelpText(&output)
	})

	Context("when any required flags are missing", func() {
		var output helpText
		var session *gexec.Session
		var command []string
		var env []string
		BeforeEach(func() {
			env = []string{"BOSH_PASSWORD=admin"}
		})
		JustBeforeEach(func() {
			session = runBinary(env, command...)
			output.output = session.Out.Contents()
		})

		Context("Missing target", func() {
			BeforeEach(func() {
				command = []string{"--username", "admin", "--password", "admin", "--deployment", "my-new-deployment", "backup"}
			})
			It("Exits with non zero", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("displays a failure message", func() {
				Expect(output.outputString()).To(ContainSubstring("--target flag is required."))
			})
			ShowsTheHelpText(&output)
		})

		Context("Missing username", func() {
			BeforeEach(func() {
				command = []string{"--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup"}
			})
			It("Exits with non zero", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("displays a failure message", func() {
				Expect(output.outputString()).To(ContainSubstring("--username flag is required."))
			})
			ShowsTheHelpText(&output)
		})

		Context("Missing password in args", func() {
			BeforeEach(func() {
				env = []string{}
				command = []string{"--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup"}
			})
			It("Exits with non zero", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("displays a failure message", func() {
				Expect(output.outputString()).To(ContainSubstring("--password flag is required."))
			})
			ShowsTheHelpText(&output)
		})

		Context("Missing deployment", func() {
			BeforeEach(func() {
				command = []string{"--username", "admin", "--password", "admin", "--target", director.URL, "backup"}
			})
			It("Exits with non zero", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("displays a failure message", func() {
				Expect(output.outputString()).To(ContainSubstring("--deployment flag is required."))
			})
			ShowsTheHelpText(&output)
		})
	})

	Context("Help", func() {
		var output helpText

		BeforeEach(func() {
			output.output = runBinary([]string{"BOSH_PASSWORD=admin"}, "--help").Out.Contents()
		})

		ShowsTheHelpText(&output)
	})

	Context("no arguments", func() {
		var output helpText

		BeforeEach(func() {
			output.output = runBinary([]string{"BOSH_PASSWORD=admin"}, "").Out.Contents()
		})

		ShowsTheHelpText(&output)
	})
})
