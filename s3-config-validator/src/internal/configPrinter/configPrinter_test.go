package configPrinter_test

import (
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/cloudfoundry/bosh-backup-and-restore/s3-config-validator/src/internal/config"
	. "github.com/cloudfoundry/bosh-backup-and-restore/s3-config-validator/src/internal/configPrinter"
)

var _ = Describe("PrintConfig", func() {

	Context("Unversioned", func() {
		validConfig := Config{
			Buckets: map[string]LiveBucket{
				"Test Resource": {
					Name:     "testName",
					Region:   "testRegion",
					ID:       "testID",
					Secret:   "testSecret",
					Endpoint: "testEndpoint",
					Backup: &BackupBucket{
						Name:   "testBackupName",
						Region: "testBackupRegion",
					},
				},
			},
		}

		validConfigWithMultipleBuckets := Config{
			Buckets: map[string]LiveBucket{
				"Test Resource": {
					Name:     "testName",
					Region:   "testRegion",
					ID:       "testID",
					Secret:   "testSecret",
					Endpoint: "testEndpoint",
					Backup: &BackupBucket{
						Name:   "testBackupName",
						Region: "testBackupRegion",
					},
				},
				"Test Resource 2": {
					Name:     "testName",
					Region:   "testRegion",
					ID:       "testID",
					Secret:   "testSecret",
					Endpoint: "testEndpoint",
					Backup: &BackupBucket{
						Name:   "testBackupName",
						Region: "testBackupRegion",
					},
				},
			},
		}

		var writer io.Writer

		BeforeEach(func() {
			writer = gbytes.NewBuffer()
		})

		When("Given an empty Config struct", func() {
			It("Prints an empty config", func() {
				PrintConfig(writer, Config{})

				Eventually(writer).Should(gbytes.Say("{}"))
			})
		})

		When("Given a non-empty Config struct", func() {
			It("Prints the config struct as prettified JSON with Configuration heading", func() {
				prettyJSONResponse := `Configuration:

  {
    "Test Resource": {
      "name": "testName",
      "region": "testRegion",
      "aws_access_key_id": "testID",
      "aws_secret_access_key": "testSecret",
      "endpoint": "testEndpoint",
      "backup": {
        "name": "testBackupName",
        "region": "testBackupRegion"
      },
      "use_iam_profile": false
    }
  }`
				PrintConfig(writer, validConfig)

				Eventually(writer).Should(gbytes.Say(prettyJSONResponse))
			})

			It("Prints all the buckets in the config with the Configuration heading", func() {
				prettyJSONResponse := `Configuration:

  {
    "Test Resource": {
      "name": "testName",
      "region": "testRegion",
      "aws_access_key_id": "testID",
      "aws_secret_access_key": "testSecret",
      "endpoint": "testEndpoint",
      "backup": {
        "name": "testBackupName",
        "region": "testBackupRegion"
      },
      "use_iam_profile": false
    },
    "Test Resource 2": {
      "name": "testName",
      "region": "testRegion",
      "aws_access_key_id": "testID",
      "aws_secret_access_key": "testSecret",
      "endpoint": "testEndpoint",
      "backup": {
        "name": "testBackupName",
        "region": "testBackupRegion"
      },
      "use_iam_profile": false
    }
  }`
				PrintConfig(writer, validConfigWithMultipleBuckets)

				Eventually(writer).Should(gbytes.Say(prettyJSONResponse))
			})
		})

	})

	Context("Versioned", func() {
		validConfig := Config{
			Buckets: map[string]LiveBucket{
				"Test Resource": {
					Name:     "testName",
					Region:   "testRegion",
					ID:       "testID",
					Secret:   "testSecret",
					Endpoint: "testEndpoint",
				},
			},
		}

		validConfigWithMultipleBuckets := Config{
			Buckets: map[string]LiveBucket{
				"Test Resource": {
					Name:     "testName",
					Region:   "testRegion",
					ID:       "testID",
					Secret:   "testSecret",
					Endpoint: "testEndpoint",
				},
				"Test Resource 2": {
					Name:     "testName",
					Region:   "testRegion",
					ID:       "testID",
					Secret:   "testSecret",
					Endpoint: "testEndpoint",
				},
			},
		}

		var writer io.Writer

		BeforeEach(func() {
			writer = gbytes.NewBuffer()
		})

		When("Given an empty Config struct", func() {
			It("Prints an empty config", func() {
				PrintConfig(writer, Config{})

				Eventually(writer).Should(gbytes.Say("{}"))
			})
		})

		When("Given a non-empty Config struct", func() {
			It("Prints the config struct as prettified JSON with Configuration heading", func() {
				prettyJSONResponse := `Configuration:

  {
    "Test Resource": {
      "name": "testName",
      "region": "testRegion",
      "aws_access_key_id": "testID",
      "aws_secret_access_key": "testSecret",
      "endpoint": "testEndpoint",
      "use_iam_profile": false
    }
  }`
				PrintConfig(writer, validConfig)

				Eventually(writer).Should(gbytes.Say(prettyJSONResponse))
			})

			It("Prints all the buckets in the config with the Configuration heading", func() {
				prettyJSONResponse := `Configuration:

  {
    "Test Resource": {
      "name": "testName",
      "region": "testRegion",
      "aws_access_key_id": "testID",
      "aws_secret_access_key": "testSecret",
      "endpoint": "testEndpoint",
      "use_iam_profile": false
    },
    "Test Resource 2": {
      "name": "testName",
      "region": "testRegion",
      "aws_access_key_id": "testID",
      "aws_secret_access_key": "testSecret",
      "endpoint": "testEndpoint",
      "use_iam_profile": false
    }
  }`
				PrintConfig(writer, validConfigWithMultipleBuckets)

				Eventually(writer).Should(gbytes.Say(prettyJSONResponse))
			})
		})

	})

})
