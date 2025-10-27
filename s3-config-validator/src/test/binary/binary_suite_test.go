package binary_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestBinary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Binary Suite")
}

var (
	binary                     string
	awsAccessKey               string
	awsSecretKey               string
	awsAssumeRoleARN           string
	validUnversionedConfigFile *os.File
	validVersionedConfigFile   *os.File
)

var _ = BeforeSuite(func() {
	var err error
	binary, err = gexec.Build("github.com/cloudfoundry/bosh-backup-and-restore/s3-config-validator/src/cmd")
	Expect(err).NotTo(HaveOccurred())

	checkRequiredEnvs([]string{
		"AWS_ACCESS_KEY",
		"AWS_SECRET_KEY",
		"AWS_ASSUMED_ROLE_ARN",
	})

	awsAccessKey = os.Getenv("AWS_ACCESS_KEY")
	awsSecretKey = os.Getenv("AWS_SECRET_KEY")
	awsAssumeRoleARN = os.Getenv("AWS_ASSUMED_ROLE_ARN")

	validVersionedConfigFile = createVersionedConfigFile("bbr-s3-validator-versioned-bucket", awsAccessKey, awsSecretKey, awsAssumeRoleARN, "eu-west-1")
	validUnversionedConfigFile = createUnversionedConfigFile("bbr-s3-validator-e2e-all-permissions", awsAccessKey, awsSecretKey, awsAssumeRoleARN, "eu-west-1", "eu-west-1")
})

var _ = AfterSuite(func() {
	err := os.Remove(validUnversionedConfigFile.Name())

	if err != nil {
		Fail(err.Error())
	}

	err = os.Remove(validVersionedConfigFile.Name())

	if err != nil {
		Fail(err.Error())
	}
})

func checkRequiredEnvs(envs []string) {
	for _, env := range envs {
		_, present := os.LookupEnv(env)

		if !present {
			_, _ = fmt.Fprintf(os.Stderr, "Environment Variable %s must be set", env)
			os.Exit(1)
		}
	}
}

func createUnversionedConfigFile(bucketName, awsAccessKey, awsSecretKey, awsAssumeRoleARN, liveRegion, backupRegion string) *os.File {
	configFile, err := os.CreateTemp("/tmp", "bbr_s3_validator_e2e")
	Expect(err).NotTo(HaveOccurred())

	fileContents := fmt.Sprintf(`
	{
		"test-resource": {
			"aws_access_key_id": "%[2]s",
			"aws_secret_access_key": "%[3]s",
			"aws_assumed_role_arn": "%[4]s",
			"endpoint": "",
			"name": "%[1]s",
			"region": "%[5]s",
			"backup": {
				"name": "%[1]s",
				"region": "%[6]s"
			}
		}
	}
	`, bucketName, awsAccessKey, awsSecretKey, awsAssumeRoleARN, liveRegion, backupRegion)

	_, err = configFile.WriteString(fileContents)
	Expect(err).NotTo(HaveOccurred())

	return configFile
}

func createVersionedConfigFile(bucketName, awsAccessKey, awsSecretKey, awsAssumeRoleARN, liveRegion string) *os.File {
	configFile, err := os.CreateTemp("/tmp", "bbr_s3_validator_e2e")
	Expect(err).NotTo(HaveOccurred())

	fileContents := fmt.Sprintf(`
	{
		"test-resource": {
			"aws_access_key_id": "%[2]s",
			"aws_secret_access_key": "%[3]s",
			"aws_assumed_role_arn": "%[4]s",
			"endpoint": "",
			"name": "%[1]s",
			"region": "%[5]s"
		}
	}
	`, bucketName, awsAccessKey, awsSecretKey, awsAssumeRoleARN, liveRegion)

	_, err = configFile.WriteString(fileContents)
	Expect(err).NotTo(HaveOccurred())

	return configFile
}
