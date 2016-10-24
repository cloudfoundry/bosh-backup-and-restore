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

	fmt.Println(command)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

	Expect(err).ToNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
}

func RunCommandOnRemote(cmd string, remoteComand string) *gexec.Session {
	cmdParts := strings.Split(cmd, " ")
	commandPath := cmdParts[0]
	combinedArgs := append(cmdParts[1:], remoteComand)
	command := exec.Command(commandPath, combinedArgs...)

	fmt.Println(command)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

	Expect(err).ToNot(HaveOccurred())
	return session
}

func GenericBoshCommand() string {
	return fmt.Sprintf("bosh-cli --non-interactive --environment=%s --ca-cert=%s --user=%s --password=%s",
		MustHaveEnv("BOSH_URL"),
		MustHaveEnv("BOSH_CERT_PATH"),
		MustHaveEnv("BOSH_USER"),
		MustHaveEnv("BOSH_PASSWORD"),
	)
}

func TestDeploymentBoshCommand() string {
	return fmt.Sprintf("%s --deployment=%s", GenericBoshCommand(), TestDeployment())
}

func TestDeployment() string {
	return "systest-" + TestEnv()
}
func JumpboxDeployment() string {
	return "jumpbox-" + TestEnv()
}

func TestDeploymentManifest() string {
	return "../fixtures/" + TestDeployment() + ".yml"
}
func JumpboxDeploymentManifest() string {
	return "../fixtures/" + JumpboxDeployment() + ".yml"
}

func TestEnv() string {
	return MustHaveEnv("TEST_ENV")
}

func JumpBoxBoshCommand() string {
	return fmt.Sprintf("%s --deployment=%s", GenericBoshCommand(), JumpboxDeployment())
}

func JumpBoxSCPCommand() string {
	fmt.Println("$$$$")
	fmt.Printf("%s scp --gw-user=%s --gw-host=%s", JumpBoxBoshCommand(), MustHaveEnv("BOSH_GATEWAY_USER"), MustHaveEnv("BOSH_GATEWAY_HOST"))
	fmt.Println("$$$$")
	return fmt.Sprintf("%s scp --gw-user=%s --gw-host=%s", JumpBoxBoshCommand(), MustHaveEnv("BOSH_GATEWAY_USER"), MustHaveEnv("BOSH_GATEWAY_HOST"))
}
func JumpBoxSSHCommand() string {
	return fmt.Sprintf("%s ssh --gw-user=%s --gw-host=%s jumpbox/0", JumpBoxBoshCommand(), MustHaveEnv("BOSH_GATEWAY_USER"), MustHaveEnv("BOSH_GATEWAY_HOST"))
}
