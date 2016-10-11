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
		director = mockbosh.New()
		director.ExpectedBasicAuth("admin", "admin")
	})

	Context("Params", func() {
		It("can invoke command with short names", func() {
			director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").NotFound())

			runBinary([]string{}, "-u", "admin", "-p", "admin", "-t", director.URL, "-d", "my-new-deployment", "backup")

			director.VerifyMocks()
		})
		It("can invoke command with long names", func() {
			director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").NotFound())

			runBinary([]string{}, "--username", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")

			director.VerifyMocks()
		})
	})
	Context("password is supported from env", func() {
		It("can invoke command with long names", func() {
			director.VerifyAndMock(mockbosh.GetDeployment("my-new-deployment").NotFound())

			runBinary([]string{"BOSH_PASSWORD=admin"}, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")

			director.VerifyMocks()
		})
	})

	Context("Wrong global args", func() {
		var output helpText
		var session *gexec.Session
		BeforeEach(func() {
			session = runBinary([]string{"BOSH_PASSWORD=admin"}, "--dave", "admin", "--password", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")
			output.output = session.Out.Contents()
		})

		It("Exists with non zero", func() {
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
			It("Exists with non zero", func() {
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
			It("Exists with non zero", func() {
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
			It("Exists with non zero", func() {
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
			It("Exists with non zero", func() {
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
