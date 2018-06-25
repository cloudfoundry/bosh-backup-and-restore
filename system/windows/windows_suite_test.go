package windows

import (
	"fmt"
	"time"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	commandPath              string
	err                      error
	JumpboxWindowsInstance   = JumpboxWindowsDeployment.Instance("jumpbox", "0")
	JumpboxWindowsDeployment = DeploymentWithName("jumpbox-windows")
	RedisWindowsDeployment   = DeploymentWithName("redis-windows")
	fixturesPath             = "../../fixtures/redis-backup/"
	workspaceDir             = "/var/vcap/store/bbr-backup_workspace"
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(15 * time.Minute)

	By("building bbr")
	commandPath, err = gexec.BuildWithEnvironment("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr", []string{"GOOS=linux", "GOARCH=amd64"})
	Expect(err).NotTo(HaveOccurred())

	By("setting up the jump box")
	Eventually(JumpboxWindowsInstance.RunCommand(
		fmt.Sprintf("sudo mkdir %s && sudo chown vcap:vcap %s && sudo chmod 0777 %s", workspaceDir, workspaceDir, workspaceDir))).Should(gexec.Exit(0))

	By("writing $BOSH_CA_CERT to a temp file")
	boshCaCertPath, err := WriteEnvVarToTempFile("BOSH_CA_CERT")
	Expect(err).NotTo(HaveOccurred())

	By("copying bbr and bosh.crt to the jumpbox")
	JumpboxWindowsInstance.Copy(commandPath, workspaceDir)
	JumpboxWindowsInstance.Copy(boshCaCertPath, workspaceDir+"/bosh.crt")
})

var _ = AfterSuite(func() {
	By("cleaning up the jumpbox")
	command := fmt.Sprintf("sudo rm -rf %s", workspaceDir)
	Eventually(JumpboxWindowsInstance.RunCommand(command)).Should(gexec.Exit(0))
})

func TestSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Windows system Suite")
}
