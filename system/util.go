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
	return fmt.Sprintf("bosh-cli --non-interactive --environment=%s --ca-cert=%s --user=%s --password=%s",
		MustHaveEnv("BOSH_URL"),
		MustHaveEnv("BOSH_CERT_PATH"),
		MustHaveEnv("BOSH_USER"),
		MustHaveEnv("BOSH_PASSWORD"),
	)
}

func RedisDeploymentBoshCommand() string {
	return fmt.Sprintf("%s --deployment=%s", GenericBoshCommand(), RedisDeployment())
}

func AnotherRedisDeploymentBoshCommand() string {
	return fmt.Sprintf("%s --deployment=%s", GenericBoshCommand(), AnotherRedisDeployment())
}

func RedisDeploymentSCPCommand() string {
	return fmt.Sprintf("%s scp --gw-user=%s --gw-host=%s --gw-private-key=%s", RedisDeploymentBoshCommand(), MustHaveEnv("BOSH_GATEWAY_USER"), MustHaveEnv("BOSH_GATEWAY_HOST"), MustHaveEnv("BOSH_GATEWAY_KEY"))
}
func RedisDeploymentSSHCommand(instanceName, instanceIndex string) string {
	return fmt.Sprintf(
		"%s ssh --gw-user=%s --gw-host=%s --gw-private-key=%s %s/%s",
		RedisDeploymentBoshCommand(),
		MustHaveEnv("BOSH_GATEWAY_USER"),
		MustHaveEnv("BOSH_GATEWAY_HOST"),
		MustHaveEnv("BOSH_GATEWAY_KEY"),
		instanceName,
		instanceIndex,
	)
}

func RedisDeployment() string {
	return "redis-" + TestEnv()
}

func AnotherRedisDeployment() string {
	return "another-redis-" + TestEnv()
}

func JumpboxDeployment() string {
	return "jumpbox-" + TestEnv()
}

func RedisDeploymentManifest() string {
	return "../fixtures/redis.yml"
}

func AnotherRedisDeploymentManifest() string {
	return "../fixtures/another-redis.yml"
}

func JumpboxDeploymentManifest() string {
	return "../fixtures/jumpbox.yml"
}

func TestEnv() string {
	return MustHaveEnv("TEST_ENV")
}

func JumpBoxBoshCommand() string {
	return fmt.Sprintf("%s --deployment=%s", GenericBoshCommand(), JumpboxDeployment())
}

func JumpBoxSCPCommand() string {
	return fmt.Sprintf("%s scp --gw-user=%s --gw-host=%s --gw-private-key=%s", JumpBoxBoshCommand(), MustHaveEnv("BOSH_GATEWAY_USER"), MustHaveEnv("BOSH_GATEWAY_HOST"), MustHaveEnv("BOSH_GATEWAY_KEY"))
}
func JumpBoxSSHCommand() string {
	return fmt.Sprintf("%s ssh --gw-user=%s --gw-host=%s --gw-private-key=%s jumpbox/0", JumpBoxBoshCommand(), MustHaveEnv("BOSH_GATEWAY_USER"), MustHaveEnv("BOSH_GATEWAY_HOST"), MustHaveEnv("BOSH_GATEWAY_KEY"))
}
