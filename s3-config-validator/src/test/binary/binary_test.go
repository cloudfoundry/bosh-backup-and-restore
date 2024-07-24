package binary_test

import (
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const ConfigPathEnv = "BBR_S3_BUCKETS_CONFIG"

var _ = Describe("binary tests", func() {
	var session *gexec.Session

	Describe("Config file reading", func() {

		AfterEach(func() {
			os.Unsetenv(ConfigPathEnv)
		})

		When("BBR_S3_BUCKETS_CONFIG is set", func() {
			Context("file exists but is invalid", func() {

				BeforeEach(func() {
					testConfigFile := createFile(``)
					defer deleteFile(testConfigFile)

					session = executeBBRValidatorUnversioned(testConfigFile)
				})

				Context("is not valid", func() {
					It("fails with an error message", func() {
						Eventually(session, "60s").Should(gexec.Exit(1))
						Eventually(session.Out).Should(gbytes.Say(`Bad config`))
					})
				})
			})

			Context("file does not exist", func() {

				BeforeEach(func() {
					session = executeBBRValidatorUnversioned("file/does/not.exist")
				})

				It("fails with an error message", func() {
					Eventually(session).Should(gexec.Exit(1))
					Eventually(session.Out).Should(gbytes.Say(`no such file`))
				})
			})
		})

		When("BBR_S3_BUCKETS_CONFIG is unset", func() {
			Context("with no flags", func() {
				When("there is no file at default location", func() {

					BeforeEach(func() {
						os.Unsetenv(ConfigPathEnv)
						session = executeBBRValidatorVersioned("")
					})

					It("fails with an error message", func() {
						Eventually(session).Should(gexec.Exit(1))
						Eventually(session.Out).Should(gbytes.Say(
							`open /var/vcap/jobs/s3-versioned-blobstore-backup-restorer/config/buckets.json: no such file`))
					})
				})
			})

			Context("with --unversioned", func() {
				When("there is no file at default location", func() {

					BeforeEach(func() {
						os.Unsetenv(ConfigPathEnv)
						session = executeBBRValidatorUnversioned("")
					})

					It("fails with an error message", func() {
						Eventually(session).Should(gexec.Exit(1))
						Eventually(session.Out).Should(gbytes.Say(
							`open /var/vcap/jobs/s3-unversioned-blobstore-backup-restorer/config/buckets.json: no such file`))
					})
				})

			})
		})
	})

	Describe("Validation when run with", func() {
		Context("no flags", func() {

			BeforeEach(func() {
				session = executeBBRValidatorVersioned(validVersionedConfigFile.Name())
			})

			It("displays general information", func() {
				Eventually(session, "20s").Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
				Make sure to run this on your 'backup & restore' VM.

				Validating versioned S3 buckets configuration at:

				  ` + validVersionedConfigFile.Name())))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
	Configuration:

	  {`)))
			})

			It("successfully validates all operations", func() {
				Eventually(session, "20s").Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
					Validating test-resource's live bucket bbr-s3-validator-versioned-bucket ...
					 * Bucket is versioned ... Yes
					 * Can list object versions ... Yes
					 * Can get object versions ... Yes
			
					Good config
			
					Run with --validate-put-object to test writing objects to the buckets. Disclaimer: This will write test files to the buckets.
				`)))
			})
		})

		Context("with --validate-put-object", func() {

			BeforeEach(func() {
				session = executeBBRValidatorVersioned(validVersionedConfigFile.Name(), "--validate-put-object")
			})

			It("displays general information", func() {
				Eventually(session, "20s").Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
				Make sure to run this on your 'backup & restore' VM.

				Validating versioned S3 buckets configuration at:

				  ` + validVersionedConfigFile.Name())))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
	Configuration:

	  {`)))
			})

			It("successfully validates all operations", func() {
				Eventually(session, "20s").Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
					Validating test-resource's live bucket bbr-s3-validator-versioned-bucket ...
					 * Bucket is versioned ... Yes
					 * Can list object versions ... Yes
					 * Can get object versions ... Yes
					 * Can put objects ... Yes
					
					Good config
				`)))
			})
		})

		Context("with --unversioned", func() {

			BeforeEach(func() {
				session = executeBBRValidatorUnversioned(validUnversionedConfigFile.Name())
			})

			It("displays general information", func() {
				Eventually(session, "60s").Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
				Make sure to run this on your 'backup & restore' VM.

				Validating unversioned S3 buckets configuration at:

				  ` + validUnversionedConfigFile.Name())))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
	Configuration:

	  {`)))
			})

			It("successfully validates just read-only operations", func() {
				session := executeBBRValidatorUnversioned(validUnversionedConfigFile.Name())

				Eventually(session, "60s").Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
					Validating test-resource's live bucket bbr-s3-validator-e2e-all-permissions ...
					 * Bucket is not versioned ... Yes
					 * Can list objects ... Yes
					 * Can get objects ... Yes

					Validating test-resource's backup bucket bbr-s3-validator-e2e-all-permissions ...
					 * Bucket is not versioned ... Yes
					 * Can list objects ... Yes
					 * Can get objects ... Yes
					
					Good config

					Run with --validate-put-object to test writing objects to the buckets. Disclaimer: This will write test files to the buckets.
					`)))
			})
		})

		Context("with --unversioned && --validate-put-object", func() {

			BeforeEach(func() {
				session = executeBBRValidatorUnversioned(validUnversionedConfigFile.Name(), "--validate-put-object")
			})

			It("displays general information", func() {
				Eventually(session, "60s").Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
				Make sure to run this on your 'backup & restore' VM.

				Validating unversioned S3 buckets configuration at:

				  ` + validUnversionedConfigFile.Name())))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
	Configuration:

	  {`)))
			})

			It("successfully validates all operations", func() {
				Eventually(session, "60s").Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
					Validating test-resource's live bucket bbr-s3-validator-e2e-all-permissions ...
					 * Bucket is not versioned ... Yes
					 * Can list objects ... Yes
					 * Can get objects ... Yes
					 * Can put objects ... Yes

					Validating test-resource's backup bucket bbr-s3-validator-e2e-all-permissions ...
					 * Bucket is not versioned ... Yes
					 * Can list objects ... Yes
					 * Can get objects ... Yes
					 * Can put objects ... Yes
					
					Good config
				`)))
			})
		})
	})

	Describe("Help messaging", func() {
		When("the binary is invoked with --help", func() {

			BeforeEach(func() {
				session = executeBBRValidatorUnversioned("", "--help")
			})

			It("Displays the usage", func() {
				Eventually(session, "60s").Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(dedent(`
					Validates a BOSH backup and restore bucket configuration.
					By default it will assume versioned buckets unless specified otherwise.
					
					The default config file locations are:
					
					 * versioned: /var/vcap/jobs/s3-versioned-blobstore-backup-restorer/config/buckets.json
					 * unversioned: /var/vcap/jobs/s3-unversioned-blobstore-backup-restorer/config/buckets.json

					Make sure to run this on the ‘backup_restore’ VM.
					
					USAGE:
					  bbr-s3-config-validator [--validate-put-object]
					
					OPTIONS:
					  --help                        Show usage.
					  --unversioned                 Validate unversioned bucket configuration.
					  --validate-put-object         Test writing objects to the buckets. Disclaimer: This will write test files to the buckets.
					
					ENVIRONMENT VARIABLES:
					  BBR_S3_BUCKETS_CONFIG=<path>  Override the default bucket configuration file location
					`)))
			})
		})
	})

})

func dedent(text string) string {
	return strings.ReplaceAll(strings.TrimSpace(text), "	", "")
}

func createFile(content string) string {
	testConfigFile, _ := os.CreateTemp("/tmp", "test_config.json")

	_, err := testConfigFile.WriteString(content)
	Expect(err).NotTo(HaveOccurred())

	return testConfigFile.Name()
}

func deleteFile(filePath string) {
	os.Remove(filePath)
}

func executeBBRValidatorUnversioned(configFilePath string, args ...string) *gexec.Session {
	os.Setenv("BBR_S3_BUCKETS_CONFIG", configFilePath)

	args = append(args, "--unversioned")

	command := exec.Command(binary, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

	Expect(err).NotTo(HaveOccurred())

	return session
}

func executeBBRValidatorVersioned(configFilePath string, args ...string) *gexec.Session {
	os.Setenv("BBR_S3_BUCKETS_CONFIG", configFilePath)

	command := exec.Command(binary, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

	Expect(err).NotTo(HaveOccurred())

	return session
}
