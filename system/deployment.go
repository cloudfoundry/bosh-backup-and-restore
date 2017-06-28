package system

import (
	"os/exec"
	"strings"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type Deployment struct {
	Name     string
	Manifest string
}

func NewDeployment(name, manifest string) Deployment {
	return Deployment{Name: name, Manifest: manifest}
}

func (d Deployment) Deploy() {
	session := d.runBosh("deploy", "--var=deployment-name="+d.Name, d.Manifest)
	Eventually(session).Should(gexec.Exit(0))
}

func (d Deployment) Delete() {
	session := d.runBosh("delete-deployment")
	Eventually(session).Should(gexec.Exit(0))
}

func (d Deployment) RunCommand(instanceName, instanceIndex, command string) *gexec.Session {
	return d.runBosh("ssh",
		"--gw-user="+MustHaveEnv("BOSH_GATEWAY_USER"),
		"--gw-host="+MustHaveEnv("BOSH_GATEWAY_HOST"),
		"--gw-private-key="+MustHaveEnv("BOSH_GATEWAY_KEY"),
		instanceName+"/"+instanceIndex,
		command)
}

func (d Deployment) RunCommandAs(user, instanceName, instanceIndex, command string) *gexec.Session {
	return d.RunCommand(instanceName, instanceIndex, fmt.Sprintf("sudo su vcap -c '%s'", command))
}

func (d Deployment) Copy(instanceName, instanceIndex, sourcePath, destinationPath string) {
	session := d.runBosh("scp",
		"--gw-user="+MustHaveEnv("BOSH_GATEWAY_USER"),
		"--gw-host="+MustHaveEnv("BOSH_GATEWAY_HOST"),
		"--gw-private-key="+MustHaveEnv("BOSH_GATEWAY_KEY"),
		sourcePath,
		instanceName+"/"+instanceIndex+":"+destinationPath,
	)
	Eventually(session).Should(gexec.Exit(0))
}

func (d Deployment) runBosh(args ...string) *gexec.Session {
	boshCommand := fmt.Sprintf("bosh-cli --non-interactive --environment=%s --deployment=%s --ca-cert=%s --client=%s --client-secret=%s",
		MustHaveEnv("BOSH_URL"),
		d.Name,
		MustHaveEnv("BOSH_CERT_PATH"),
		MustHaveEnv("BOSH_CLIENT"),
		MustHaveEnv("BOSH_CLIENT_SECRET"),
	)

	return run(boshCommand, args...)
}

func run(cmd string, args ...string) *gexec.Session {
	cmdParts := strings.Split(cmd, " ")
	commandPath := cmdParts[0]
	combinedArgs := append(cmdParts[1:], args...)
	command := exec.Command(commandPath, combinedArgs...)

	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

	Expect(err).ToNot(HaveOccurred())
	return session
}
