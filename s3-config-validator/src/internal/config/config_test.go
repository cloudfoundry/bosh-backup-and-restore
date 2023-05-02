package config_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/config"
)

var _ = Describe("Config", func() {

	Context("Versioned", func() {
		validConfig := `{
    "buildpacks": {
        "aws_access_key_id": "test_access_key_id",
        "aws_secret_access_key": "test_secret_access_key",
        "endpoint": "test_endpoint",
        "name": "test_name",
        "region": "test_region"
    },
    "packages": {
        "aws_access_key_id": "test_access_key_id",
        "aws_secret_access_key": "test_secret_access_key",
        "endpoint": "test_endpoint",
        "name": "test_name",
        "region": "test_region"
    }
}`

		singleEmptyValueConfig := `{
    "buildpacks": {
        "aws_access_key_id": "test_aws_access_key_id",
        "aws_secret_access_key": "test_aws_secret_access_key",
        "endpoint": "test_endpoint",
        "name": "test_name"
    }
	}`

		allEmptyValueConfig := `{
    "buildpacks": {
        "aws_access_key_id": "",
        "aws_secret_access_key": "",
        "endpoint": "",
        "name": "",
        "region": ""
    }
	}`

		invalidIAMPlusCredsConfig := `{
    "buildpacks": {
        "aws_access_key_id": "test_access_key_id",
        "aws_secret_access_key": "test_secret_access_key",
        "name": "test_name",
        "region": "test_region",
        "use_iam_profile": true
    }
}`

		invalidIAMPlusEndpointConfig := `{
  "buildpacks": {
    "endpoint": "test_endpoint",
    "name": "test_name",
    "region": "test_region",
    "use_iam_profile": true
  }
}`

		Context("given a path to an existing, readable file", func() {
			Context("contents are valid", func() {
				It("reads the file contents", func() {
					filePath := CreateFile(validConfig)
					defer DeleteFile(filePath)

					conf, err := config.Read(filePath, true)

					Expect(err).NotTo(HaveOccurred())
					Expect(conf).To(Equal(config.Config{
						Buckets: map[string]config.LiveBucket{
							"buildpacks": {
								Name:     "test_name",
								Region:   "test_region",
								ID:       "test_access_key_id",
								Secret:   "test_secret_access_key",
								Endpoint: "test_endpoint",
							},
							"packages": {
								Name:     "test_name",
								Region:   "test_region",
								ID:       "test_access_key_id",
								Secret:   "test_secret_access_key",
								Endpoint: "test_endpoint",
							},
						},
					}))
				})
			})

			Context("contents are invalid", func() {
				When("given an invalid json", func() {
					It("returns an error", func() {
						testFile := CreateFile("not json")
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, true)

						Expect(err).To(HaveOccurred())
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("given an empty json", func() {
					It("returns an error", func() {
						testFile := CreateFile("{}")
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, true)

						Expect(err).To(MatchError("invalid config: json was empty"))
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("one field is empty", func() {
					It("returns an error", func() {
						testFile := CreateFile(singleEmptyValueConfig)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, true)

						Expect(err).To(MatchError("invalid config: fields [buildpacks.region] are empty\n"))
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("all fields are empty", func() {
					It("returns an error", func() {
						testFile := CreateFile(allEmptyValueConfig)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, true)

						Expect(err).To(MatchError("invalid config: fields" +
							" [buildpacks.aws_access_key_id" +
							" buildpacks.aws_secret_access_key" +
							" buildpacks.name buildpacks.region]" +
							" are empty\n"))
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("we try to use IAM and a Secret Access Key at the same time", func() {
					It("returns a helpful error", func() {
						testFile := CreateFile(invalidIAMPlusCredsConfig)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, true)

						Expect(err).To(MatchError("invalid config: because use_iam_profile is set to true, there should be no aws_access_key_id or aws_secret_access_key in the following buckets: [buildpacks]\n"))
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("we try to use IAM and an Endpoint at the same time", func() {
					It("should return a helpful error message", func() {
						testFile := CreateFile(invalidIAMPlusEndpointConfig)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, true)
						Expect(conf).To(Equal(config.Config{}))
						Expect(err).To(MatchError("invalid config: because use_iam_profile is set to true, the endpoint field must not be set in the following buckets: [buildpacks]\n"))
					})
				})
			})
		})

		Context("given a path to a file that does not exist", func() {
			It("returns an error", func() {
				conf, err := config.Read("/this/file/does/not.exist", true)

				Expect(err).To(HaveOccurred())
				Expect(conf).To(Equal(config.Config{}))
			})
		})

		Context("given a path to an existing, unreadable file", func() {
			It("returns an error", func() {
				filePath := CreateFile(validConfig)
				defer DeleteFile(filePath)

				var noRead os.FileMode = 0o300

				f, err := os.Open(filePath)
				Expect(err).NotTo(HaveOccurred())
				err = f.Chmod(noRead)
				Expect(err).NotTo(HaveOccurred())
				f.Close()

				conf, err := config.Read(filePath, true)

				Expect(err).To(HaveOccurred())
				Expect(conf).To(Equal(config.Config{}))
			})
		})

	})

	Context("Unversioned", func() {
		validConfig := `{
    "buildpacks": {
        "aws_access_key_id": "test_access_key_id",
        "aws_secret_access_key": "test_secret_access_key",
        "backup": {
            "name": "test_backup_name",
            "region": "test_backup_region"
        },
        "endpoint": "test_endpoint",
        "name": "test_name",
        "region": "test_region"
    },
    "packages": {
        "aws_access_key_id": "test_access_key_id",
        "aws_secret_access_key": "test_secret_access_key",
        "backup": {
            "name": "test_backup_name",
            "region": "test_backup_region"
        },
        "endpoint": "test_endpoint",
        "name": "test_name",
        "region": "test_region"
    }
}`

		singleEmptyValueConfig := `{
    "buildpacks": {
        "aws_access_key_id": "test_aws_access_key_id",
        "aws_secret_access_key": "test_aws_secret_access_key",
        "backup": {
            "name": "",
            "region": "test_backup_region"
        },
        "endpoint": "test_endpoint",
        "name": "test_name",
        "region": "test_region"
    }
	}`

		allEmptyValueConfig := `{
    "buildpacks": {
        "aws_access_key_id": "",
        "aws_secret_access_key": "",
        "backup": {
            "name": "",
            "region": ""
        },
        "endpoint": "",
        "name": "",
        "region": ""
    }
	}`

		invalidIAMPlusCredsConfig := `{
    "buildpacks": {
        "aws_access_key_id": "test_access_key_id",
        "aws_secret_access_key": "test_secret_access_key",
        "name": "test_name",
        "region": "test_region",
        "use_iam_profile": true,
        "backup": {
            "name": "another_test_name",
            "region": "another_test_region"
        }
    }
}`
		invalidIAMPlusEndpointConfig := `{
  "buildpacks": {
    "endpoint": "test_endpoint",
    "name": "test_name",
    "region": "test_region",
    "use_iam_profile": true,
    "backup": {
      "name": "another_test_name",
      "region": "another_test_region"
    }
  }
}`

		invalidMissingBackupConfig := `{
    "buildpacks": {
        "aws_access_key_id": "test_access_key_id",
        "aws_secret_access_key": "test_secret_access_key",
        "endpoint": "test_endpoint",
        "name": "test_name",
        "region": "test_region"
    }
}`

		configWithMultipleIssues := `{
    "bucketMissingFieldNames": {
	    "aws_access_key_id": "",
	    "aws_secret_access_key": "",
	    "backup": {
		    "name": "backupname",
		    "region": "backupregion"
	    },
	    "name": "",
	    "region": ""
    },
    "bucketMissingBackupBuckets": {
	    "aws_access_key_id": "id",
	    "aws_secret_access_key": "secret",
	    "name": "missingBackupBucketName",
	    "region": "region"
    },
    "bucketWithTooManyCreds": {
	    "aws_access_key_id": "id",
	    "aws_secret_access_key": "secret",
	    "use_iam_profile": true,
	    "backup": {
		    "name": "backupname",
		    "region": "backupregion"
	    },
	    "name": "missingBackupBucketName",
	    "region": "region"
    },
    "bucketWithAllTheProblems": {
	    "aws_access_key_id": "id",
	    "aws_secret_access_key": "secret",
	    "use_iam_profile": true,
	    "name": "",
	    "region": ""
    }
}`

		Context("given a path to an existing, readable file", func() {
			Context("contents are valid", func() {
				It("reads the file contents", func() {
					filePath := CreateFile(validConfig)
					defer DeleteFile(filePath)

					conf, err := config.Read(filePath, false)

					Expect(err).NotTo(HaveOccurred())
					Expect(conf).To(Equal(config.Config{
						Buckets: map[string]config.LiveBucket{
							"buildpacks": {
								Name:     "test_name",
								Region:   "test_region",
								ID:       "test_access_key_id",
								Secret:   "test_secret_access_key",
								Endpoint: "test_endpoint",
								Backup: &config.BackupBucket{
									Name:   "test_backup_name",
									Region: "test_backup_region",
								},
							},
							"packages": {
								Name:     "test_name",
								Region:   "test_region",
								ID:       "test_access_key_id",
								Secret:   "test_secret_access_key",
								Endpoint: "test_endpoint",
								Backup: &config.BackupBucket{
									Name:   "test_backup_name",
									Region: "test_backup_region",
								},
							},
						},
					}))
				})
			})

			Context("contents are invalid", func() {
				When("given an invalid json", func() {
					It("returns an error", func() {
						testFile := CreateFile("not json")
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, false)

						Expect(err).To(HaveOccurred())
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("given an empty json", func() {
					It("returns an error", func() {
						testFile := CreateFile("{}")
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, false)

						Expect(err).To(MatchError("invalid config: json was empty"))
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("one field is empty", func() {
					It("returns an error", func() {
						testFile := CreateFile(singleEmptyValueConfig)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, false)

						Expect(err).To(MatchError("invalid config: fields [buildpacks.backup.name] are empty\n"))
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("all fields are empty", func() {
					It("returns an error", func() {
						testFile := CreateFile(allEmptyValueConfig)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, false)

						Expect(err).To(MatchError("invalid config: fields" +
							" [buildpacks.aws_access_key_id" +
							" buildpacks.aws_secret_access_key buildpacks" +
							".backup.name buildpacks.backup.region" +
							" buildpacks.name buildpacks.region]" +
							" are empty\n"))
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("we try to use IAM and a Secret Access Key at the same time", func() {
					It("returns a helpful error", func() {
						testFile := CreateFile(invalidIAMPlusCredsConfig)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, false)

						Expect(err).To(MatchError("invalid config: because use_iam_profile is set to true, there should be no aws_access_key_id or aws_secret_access_key in the following buckets: [buildpacks]\n"))
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("we try to use IAM and supply an endpoint", func() {
					It("returns a helpful error", func() {
						testFile := CreateFile(invalidIAMPlusEndpointConfig)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, false)

						Expect(conf).To(Equal(config.Config{}))
						Expect(err).To(MatchError("invalid config: because use_iam_profile is set to true, the endpoint field must not be set in the following buckets: [buildpacks]\n"))
					})
				})

				When("our unversioned bucket config is missing the backup buckets", func() {
					It("returns a helpful error", func() {
						testFile := CreateFile(invalidMissingBackupConfig)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, false)

						Expect(err).To(MatchError("invalid config: backup buckets must be specified when taking unversioned backups. The following buckets are missing backup buckets: [buildpacks]\n"))
						Expect(conf).To(Equal(config.Config{}))
					})
				})

				When("our bucket config has a lot of problems", func() {
					It("returns helpful error messages for all the problems", func() {
						testFile := CreateFile(configWithMultipleIssues)
						defer DeleteFile(testFile)

						conf, err := config.Read(testFile, false)

						Expect(err).To(MatchError(ContainSubstring("backup buckets must be specified")))
						Expect(err).To(MatchError(ContainSubstring("because use_iam_profile is set to true")))
						Expect(err).To(MatchError(ContainSubstring("fields [bucketMissingFieldNames.aws_access_key_id bucketMissingFieldNames.aws_secret_access_key bucketMissingFieldNames.name bucketMissingFieldNames.region bucketWithAllTheProblems.name bucketWithAllTheProblems.region] are empty")))
						Expect(conf).To(Equal(config.Config{}))
					})
				})
			})
		})

		Context("given a path to a file that does not exist", func() {
			It("returns an error", func() {
				conf, err := config.Read("/this/file/does/not.exist", false)

				Expect(err).To(HaveOccurred())
				Expect(conf).To(Equal(config.Config{}))
			})
		})

		Context("given a path to an existing, unreadable file", func() {
			It("returns an error", func() {
				filePath := CreateFile(validConfig)
				defer DeleteFile(filePath)

				var noRead os.FileMode = 0o300

				f, err := os.Open(filePath)
				Expect(err).NotTo(HaveOccurred())
				err = f.Chmod(noRead)
				Expect(err).NotTo(HaveOccurred())
				f.Close()

				conf, err := config.Read(filePath, false)

				Expect(err).To(HaveOccurred())
				Expect(conf).To(Equal(config.Config{}))
			})
		})

	})

})

func CreateFile(content string) string {
	testConfigFile, _ := ioutil.TempFile("/tmp", "test_config.json")

	_, err := testConfigFile.WriteString(content)
	Expect(err).NotTo(HaveOccurred())

	return testConfigFile.Name()
}

func DeleteFile(filePath string) {
	os.Remove(filePath)
}
