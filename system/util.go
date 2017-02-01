package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func MustHaveEnv(keyname string) string {
	val := os.Getenv(keyname)
	Expect(val).NotTo(BeEmpty(), "Need "+keyname+" for the test")
	return val
}

func RunBoshCommand(cmd string, args ...string) {
	cmdParts := strings.Split(cmd, " ")
	commandPath := cmdParts[0]
	combinedArgs := append(cmdParts[1:], args...)
	command := exec.Command(commandPath, combinedArgs...)

	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

	Expect(err).ToNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
}

func RunCommandOnRemote(cmd string, remoteComand string) *gexec.Session {
	cmdParts := strings.Split(cmd, " ")
	commandPath := cmdParts[0]
	combinedArgs := append(cmdParts[1:], remoteComand)
	command := exec.Command(commandPath, combinedArgs...)

	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

	Expect(err).ToNot(HaveOccurred())
	return session
}

func RunCommandOnRemoteAsVcap(cmd string, remoteComand string) *gexec.Session {
	return RunCommandOnRemote(cmd, fmt.Sprintf("sudo su vcap -c '%s'", remoteComand))
}

func AssertJumpboxFilesExist(paths []string) {
	for _, path := range paths {
		cmd := RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf("stat %s", path),
		)
		Eventually(cmd).Should(gexec.Exit(0),
			fmt.Sprintf("File at %s not found on jumpbox\n", path))
	}
}

func GenericBoshCommand() string {
	return fmt.Sprintf("bosh-cli --non-interactive --environment=%s --ca-cert=%s --client=%s --client-secret=%s",
		MustHaveEnv("BOSH_URL"),
		MustHaveEnv("BOSH_CERT_PATH"),
		MustHaveEnv("BOSH_CLIENT"),
		MustHaveEnv("BOSH_CLIENT_SECRET"),
	)
}

func JumpBoxBoshCommand() string {
	return getBoshCommand(JumpboxDeployment)
}

func RedisDeploymentBoshCommand() string {
	return getBoshCommand(RedisDeployment)
}

func AnotherRedisDeploymentBoshCommand() string {
	return getBoshCommand(AnotherRedisDeployment)
}

func RedisWithMetadataDeploymentBoshCommand() string {
	return getBoshCommand(RedisWithMetadataDeployment)
}

func JumpBoxSCPCommand() string {
	return getSCPCommand(JumpBoxBoshCommand)
}

func RedisDeploymentSCPCommand() string {
	return getSCPCommand(RedisDeploymentBoshCommand)
}

func RedisWithMetadataDeploymentSCPCommand() string {
	return getSCPCommand(RedisWithMetadataDeploymentBoshCommand)
}

func JumpBoxSSHCommand() string {
	return getSSHCommand(JumpBoxBoshCommand, "jumpbox", "0")
}

func RedisDeploymentSSHCommand(instanceName, instanceIndex string) string {
	return getSSHCommand(RedisDeploymentBoshCommand, instanceName, instanceIndex)
}

func RedisWithMetadataDeploymentSSHCommand(instanceName, instanceIndex string) string {
	return getSSHCommand(RedisWithMetadataDeploymentBoshCommand, instanceName, instanceIndex)
}



func getSCPCommand(boshCommand func() string) string {
	return fmt.Sprintf(
		"%s scp --gw-user=%s --gw-host=%s --gw-private-key=%s",
		boshCommand(),
		MustHaveEnv("BOSH_GATEWAY_USER"),
		MustHaveEnv("BOSH_GATEWAY_HOST"),
		MustHaveEnv("BOSH_GATEWAY_KEY"),
	)
}

func getBoshCommand(deploymentName func() string) string {
	return fmt.Sprintf(
		"%s --deployment=%s",
		GenericBoshCommand(),
		deploymentName(),
	)
}

func getSSHCommand(boshCmd func() string, instanceName, instanceIndex string) string {
	return fmt.Sprintf(
		"%s ssh --gw-user=%s --gw-host=%s --gw-private-key=%s %s/%s",
		boshCmd(),
		MustHaveEnv("BOSH_GATEWAY_USER"),
		MustHaveEnv("BOSH_GATEWAY_HOST"),
		MustHaveEnv("BOSH_GATEWAY_KEY"),
		instanceName,
		instanceIndex,
	)
}

func RedisDeployment() string {
	return "redis-" + testEnv()
}

func RedisWithMetadataDeployment() string {
	return "redis-with-metadata-" + testEnv()
}

func AnotherRedisDeployment() string {
	return "another-redis-" + testEnv()
}

func JumpboxDeployment() string {
	return "jumpbox-" + testEnv()
}

func RedisDeploymentManifest() string {
	return "../fixtures/redis.yml"
}

func RedisWithMetadataDeploymentManifest() string {
	return "../fixtures/redis-with-metadata.yml"
}

func AnotherRedisDeploymentManifest() string {
	return "../fixtures/another-redis.yml"
}

func JumpboxDeploymentManifest() string {
	return "../fixtures/jumpbox.yml"
}

func testEnv() string {
	return MustHaveEnv("TEST_ENV")
}
