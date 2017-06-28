package system

import (
	"fmt"
	"os"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func MustHaveEnv(keyname string) string {
	val := os.Getenv(keyname)
	Expect(val).NotTo(BeEmpty(), "Need "+keyname+" for the test")
	return val
}

func AssertJumpboxFilesExist(paths []string) {
	for _, path := range paths {
		cmd := JumpboxDeployment().Instance("jumpbox", "0").RunCommandAs("vcap", "stat "+path)
		Eventually(cmd).Should(gexec.Exit(0), fmt.Sprintf("File at %s not found on jumpbox\n", path))
	}
}

func RedisDeployment() Deployment {
	return NewDeployment("redis-"+MustHaveEnv("TEST_ENV"), "../../fixtures/redis.yml")
}

func RedisWithMetadataDeployment() Deployment {
	return NewDeployment("redis-with-metadata-"+MustHaveEnv("TEST_ENV"), "../../fixtures/redis-with-metadata.yml")
}

func RedisWithMissingScriptDeployment() Deployment {
	return NewDeployment("redis-with-missing-script-"+MustHaveEnv("TEST_ENV"), "../../fixtures/redis-with-missing-script.yml")
}

func AnotherRedisDeployment() Deployment {
	return NewDeployment("another-redis-"+MustHaveEnv("TEST_ENV"), "../../fixtures/another-redis.yml")
}

func JumpboxDeployment() Deployment {
	return NewDeployment("jumpbox-"+MustHaveEnv("TEST_ENV"), "../../fixtures/jumpbox.yml")
}

func BackupDirWithTimestamp(deploymentName string) string {
	return fmt.Sprintf("%s_*T*Z", deploymentName)
}
