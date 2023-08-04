package deployment

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockbosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockhttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
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
		AssertDeploymentCLIBehaviour := func(cmd string, extraArgs ...string) {
			Context("params", func() {
				It("can invoke command with short names", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{},
						append([]string{
							"deployment",
							"--ca-cert", sslCertPath,
							"-u", "admin",
							"-p", "admin",
							"-t", director.URL,
							"-d", "my-new-deployment",
							cmd,
						}, extraArgs...)...)

					director.VerifyMocks()
				})

				It("can invoke command with long names", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{},
						append([]string{
							"deployment",
							"--ca-cert", sslCertPath,
							"--username", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)...)

					director.VerifyMocks()
				})

				It("can invoke command with environment variables", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{
							fmt.Sprintf("BOSH_ENVIRONMENT=%s", director.URL),
							"BOSH_CLIENT=admin",
							"BOSH_CLIENT_SECRET=admin",
							"BOSH_DEPLOYMENT=my-new-deployment",
							fmt.Sprintf("BOSH_CA_CERT=%s", sslCertPath),
						},
						append([]string{"deployment", cmd}, extraArgs...)...)

					director.VerifyMocks()
				})

				It("can invoke command with the CA_CERT environment variable", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{fmt.Sprintf("CA_CERT=%s", sslCertValue)},
						append([]string{
							"deployment",
							"--username", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)...)

					director.VerifyMocks()
				})

				It("can invoke command and CA_CERT takes precedence over BOSH_CA_CERT environment variable", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{
							fmt.Sprintf("CA_CERT=%s", sslCertPath),
							"BOSH_CA_CERT=/tmp/no-cert-here",
						},
						append([]string{
							"deployment",
							"--username", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)...)

					director.VerifyMocks()
				})
			})

			Context("password is supported from env", func() {
				It("can invoke command with long names", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					binary.Run(backupWorkspace,
						[]string{"BOSH_CLIENT_SECRET=admin"},
						append([]string{
							"deployment",
							"--ca-cert", sslCertPath,
							"--username", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)...)

					director.VerifyMocks()
				})
			})

			Context("Hostname is malformed", func() {
				var session *gexec.Session
				BeforeEach(func() {
					badDirectorURL := "https://:25555"
					session = binary.Run(backupWorkspace,
						[]string{"BOSH_CLIENT_SECRET=admin"},
						append([]string{
							"deployment",
							"--ca-cert", sslCertPath,
							"--username", "admin",
							"--password", "admin",
							"--target", badDirectorURL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)...)
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(session.Err).To(gbytes.Say("invalid bosh URL"))
				})
			})

			Context("Custom CA cert cannot be read", func() {
				var session *gexec.Session
				BeforeEach(func() {
					session = binary.Run(backupWorkspace,
						[]string{"BOSH_CLIENT_SECRET=admin"},
						append([]string{
							"deployment",
							"--ca-cert", "/tmp/whatever",
							"--username", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)...)
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(session.Err).To(gbytes.Say("open /tmp/whatever: no such file or directory"))
				})
			})

			Context("Custom CA cert value is invalid", func() {
				var session *gexec.Session
				BeforeEach(func() {
					session = binary.Run(backupWorkspace,
						[]string{"BOSH_CLIENT_SECRET=admin"},
						append([]string{
							"deployment",
							"--ca-cert", "-----BEGIN",
							"--username", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)...)
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(session.Err).To(gbytes.Say("Missing PEM block"))
				})
			})

			Context("Wrong global args", func() {
				var session *gexec.Session
				BeforeEach(func() {
					session = binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"},
						append([]string{
							"deployment",
							"--dave", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)...)
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(session.Out).To(gbytes.Say("Incorrect Usage"))
				})

				It("displays the usable flags", func() {
					assertDeploymentHelpText(session)
				})
			})

			Context("when any required flags are missing", func() {
				var session *gexec.Session
				var command []string
				var env []string
				BeforeEach(func() {
					env = []string{"BOSH_CLIENT_SECRET=admin"}
				})
				JustBeforeEach(func() {
					session = binary.Run(backupWorkspace, env, command...)
				})

				Context("Missing target", func() {
					BeforeEach(func() {
						command = append([]string{
							"deployment",
							"--username", "admin",
							"--password", "admin",
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)
					})
					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err).To(gbytes.Say("--target flag is required."))
					})

					It("displays the usable flags", func() {
						assertDeploymentHelpText(session)
					})
				})

				Context("Missing username", func() {
					BeforeEach(func() {
						command = append([]string{
							"deployment",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)
					})

					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err).To(gbytes.Say("--username flag is required."))
					})

					It("displays the usable flags", func() {
						assertDeploymentHelpText(session)
					})
				})

				Context("Missing password in args", func() {
					BeforeEach(func() {
						env = []string{}
						command = append([]string{
							"deployment",
							"--username", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd,
						}, extraArgs...)
					})

					It("Exits with non zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					It("displays a failure message", func() {
						Expect(session.Err).To(gbytes.Say("--password flag is required."))
					})

					It("displays the usable flags", func() {
						assertDeploymentHelpText(session)
					})
				})
			})

			Context("with debug flag set", func() {
				It("outputs verbose HTTP logs", func() {
					director.VerifyAndMock(
						mockbosh.Info().WithAuthTypeBasic(),
						mockbosh.VMsForDeployment("my-new-deployment").NotFound(),
					)

					session := binary.Run(backupWorkspace, []string{},
						append([]string{
							"deployment",
							"--debug",
							"--ca-cert", sslCertPath,
							"--username", "admin",
							"--password", "admin",
							"--target", director.URL,
							"--deployment", "my-new-deployment",
							cmd}, extraArgs...)...)

					Expect(session.Out).To(gbytes.Say("Sending GET request to endpoint"))

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

			AssertDeploymentCLIBehaviour("restore", "--artifact-path", "my-new-deployment")

			Context("when artifact-path is not specified", func() {
				var session *gexec.Session

				BeforeEach(func() {
					session = binary.Run(backupWorkspace, []string{},
						"deployment",
						"--ca-cert", sslCertPath,
						"--username", "admin",
						"--password", "admin",
						"--target", director.URL,
						"--deployment", "my-new-deployment",
						"restore")
					Eventually(session).Should(gexec.Exit())
				})

				It("Exits with non zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("displays a failure message", func() {
					Expect(session.Err).To(gbytes.Say("--artifact-path flag is required"))
				})
			})
		})

		Context("--help", func() {
			It("displays the usable flags", func() {
				session := binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment", "--help")
				assertDeploymentHelpText(session)
			})
		})

		Context("given both --deployment and --all-deployments are provided", func() {
			var session *gexec.Session

			BeforeEach(func() {
				session = binary.Run(backupWorkspace, []string{},
					"deployment",
					"--ca-cert", sslCertPath,
					"--username", "admin",
					"--password", "admin",
					"--target", director.URL,
					"--deployment", "my-new-deployment",
					"--all-deployments",
					"backup")
				Eventually(session).Should(gexec.Exit())
			})
			It("exits non-zero", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("displays a failure message", func() {
				Expect(session.Err).To(gbytes.Say("provide one of '--deployment' or '--all-deployments' flags."))
			})
		})

		Context("given neither --deployment and --all-deployments are provided", func() {
			var session *gexec.Session

			BeforeEach(func() {
				session = binary.Run(backupWorkspace, []string{},
					"deployment",
					"--ca-cert", sslCertPath,
					"--username", "admin",
					"--password", "admin",
					"--target", director.URL,
					"backup")
				Eventually(session).Should(gexec.Exit())
			})

			It("exits non-zero", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("displays a failure message", func() {
				Expect(session.Err).To(gbytes.Say("provide one of '--deployment' or '--all-deployments' flags."))
			})
		})

		Context("no arguments", func() {
			It("displays the usable flags", func() {
				session := binary.Run(backupWorkspace, []string{"BOSH_CLIENT_SECRET=admin"}, "deployment")
				assertDeploymentHelpText(session)
			})
		})
	})
})

func assertDeploymentHelpText(session *gexec.Session) {
	Expect(session.Out).To(SatisfyAll(
		gbytes.Say("--target"), gbytes.Say("BOSH Director URL"), gbytes.Say("BOSH_ENVIRONMENT"),
		gbytes.Say("--username"), gbytes.Say("BOSH Director username"), gbytes.Say("BOSH_CLIENT"),
		gbytes.Say("--password"), gbytes.Say("BOSH Director password"), gbytes.Say("BOSH_CLIENT_SECRET"),
		gbytes.Say("--deployment"), gbytes.Say("Name of BOSH deployment. Omit if '--all-deployments' is provided"), gbytes.Say("BOSH_DEPLOYMENT"),
		gbytes.Say("--ca-cert"), gbytes.Say("Path or value of BOSH Director custom CA certificate"), gbytes.Say("CA_CERT"), gbytes.Say("BOSH_CA_CERT"),
		gbytes.Say("--debug"), gbytes.Say("Enable debug logs"),
		gbytes.Say("--all-deployments"), gbytes.Say("Run command for all deployments. Omit if '--deployment' is provided. Currently only supported for: pre-backup-check, backup and backup-cleanup"),
	))
}
