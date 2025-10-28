package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"github.com/onsi/gomega/gexec"
)

const BOSH_CLI_EXECUTABLE_NAME = "bosh"

type Deployment struct {
	Name     string
	Manifest string
}

type Instance struct {
	deployment Deployment
	Group      string
	Index      string
}

func NewDeployment(name, manifest string) Deployment {
	return Deployment{Name: name, Manifest: manifest}
}

func (d Deployment) Deploy() {
	session := d.runBosh("deploy", "--var=deployment-name="+d.Name, d.Manifest)
	EventuallyWithOffset(1, session).Should(gexec.Exit(0))
}

func (d Deployment) Delete() {
	session := d.runBosh("delete-deployment")
	EventuallyWithOffset(1, session).Should(gexec.Exit(0))
}

func (d Deployment) Instance(group, index string) Instance {
	return Instance{deployment: d, Group: group, Index: index}
}

func (d Deployment) runBosh(args ...string) *gexec.Session {
	MustHaveEnv("BOSH_ENVIRONMENT")
	MustHaveEnv("BOSH_CLIENT")
	MustHaveEnv("BOSH_CLIENT_SECRET")
	MustHaveEnv("BOSH_CA_CERT")

	boshCommand := fmt.Sprintf("%s --non-interactive --deployment=%s", BOSH_CLI_EXECUTABLE_NAME, d.Name)
	return run(boshCommand, args...)
}

func run(cmd string, args ...string) *gexec.Session {
	cmdParts := strings.Split(cmd, " ")
	commandPath := cmdParts[0]
	combinedArgs := append(cmdParts[1:], args...)
	command := exec.Command(commandPath, combinedArgs...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return session
}

func (i Instance) RunCommand(command string) *gexec.Session {
	if os.Getenv("BOSH_ALL_PROXY") == "" {
		MustHaveEnv("BOSH_GW_HOST")
		MustHaveEnv("BOSH_GW_USER")
		MustHaveEnv("BOSH_GW_PRIVATE_KEY")
	}
	return i.deployment.runBosh("ssh", i.Group+"/"+i.Index, command)
}

func (i Instance) RunCommandAs(user, command string) *gexec.Session {
	return i.RunCommand(fmt.Sprintf(`sudo su vcap -c '%s'`, command))
}

func (i Instance) Copy(sourcePath, destinationPath string) {
	if os.Getenv("BOSH_ALL_PROXY") == "" {
		MustHaveEnv("BOSH_GW_HOST")
		MustHaveEnv("BOSH_GW_USER")
		MustHaveEnv("BOSH_GW_PRIVATE_KEY")
	}

	session := i.deployment.runBosh("scp", sourcePath, i.Group+"/"+i.Index+":"+destinationPath)
	EventuallyWithOffset(1, session).Should(gexec.Exit(0))
}

func (i Instance) AssertFilesExist(paths []string) {
	for _, path := range paths {
		cmd := i.RunCommandAs("vcap", "stat "+path)
		EventuallyWithOffset(1, cmd).Should(gexec.Exit(0), fmt.Sprintf("File at %s not found on %s/%s\n", path, i.Group, i.Index))
	}
}
