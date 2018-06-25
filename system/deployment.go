package system

import (
	"os/exec"
	"strings"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"os"
)

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
	Eventually(session).Should(gexec.Exit(0))
}

func (d Deployment) Delete() {
	session := d.runBosh("delete-deployment")
	Eventually(session).Should(gexec.Exit(0))
}

func (d Deployment) Instance(group, index string) Instance {
	return Instance{deployment: d, Group: group, Index: index}
}

func (d Deployment) runBosh(args ...string) *gexec.Session {
	MustHaveEnv("BOSH_ENVIRONMENT")
	MustHaveEnv("BOSH_CLIENT")
	MustHaveEnv("BOSH_CLIENT_SECRET")
	MustHaveEnv("BOSH_CA_CERT")

	boshCommand := fmt.Sprintf("bosh-cli --non-interactive --deployment=%s", d.Name)

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
	Eventually(session).Should(gexec.Exit(0))
}

func (i Instance) AssertFilesExist(paths []string) {
	for _, path := range paths {
		cmd := i.RunCommandAs("vcap", "stat "+path)
		Eventually(cmd).Should(gexec.Exit(0), fmt.Sprintf("File at %s not found on %s/%s\n", path, i.Group, i.Index))
	}
}
