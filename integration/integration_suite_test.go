package integration

import (
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"

	"github.com/pivotal-cf/bosh-backup-and-restore/testcluster"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var commandPath string

//Default cert for golang ssh
var sslCertPath = "../../fixtures/test.crt"

var pathToPrivateKeyFile = "../../fixtures/test_rsa"
var pathToPublicKeyFile = "../fixtures/test_rsa.pub"
var runTimeout = 99999 * time.Hour

func runBinary(cwd string, env []string, params ...string) *gexec.Session {
	command := exec.Command(commandPath, params...)
	command.Env = env
	command.Dir = cwd
	fmt.Fprintf(GinkgoWriter, "Running command: %v %v in %s with env %v\n", commandPath, params, cwd, env)
	fmt.Fprintf(GinkgoWriter, "Command output start\n")
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, runTimeout).Should(gexec.Exit())
	fmt.Fprintf(GinkgoWriter, "Command output end\n")
	fmt.Fprintf(GinkgoWriter, "Exited with %d\n", session.ExitCode())

	return session
}

var _ = BeforeSuite(func() {
	var err error
	commandPath, err = gexec.Build("github.com/pivotal-cf/bosh-backup-and-restore/cmd/bbr")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	testcluster.WaitForContainersToDie()
})
