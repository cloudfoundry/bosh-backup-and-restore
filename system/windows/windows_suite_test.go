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

func TestSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Windows system Suite")
}

var (
	commandPath string
	err         error
)

var fixturesPath = "../../fixtures/redis-backup/"
var workspaceDir = "/var/vcap/store/bbr-backup_workspace"

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(15 * time.Minute)

	By("building bbr")
	commandPath, err = gexec.BuildWithEnvironment("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr", []string{"GOOS=linux", "GOARCH=amd64"})
	Expect(err).NotTo(HaveOccurred())

	By("setting up the jump box")
	Eventually(JumpboxInstance.RunCommand(
		fmt.Sprintf("sudo mkdir %s && sudo chown vcap:vcap %s && sudo chmod 0777 %s", workspaceDir, workspaceDir, workspaceDir))).Should(gexec.Exit(0))

	JumpboxInstance.Copy(commandPath, workspaceDir)
	JumpboxInstance.Copy(MustHaveEnv("BOSH_CERT_PATH"), workspaceDir+"/bosh.crt")
})

func runOnInstances(instanceCollection map[string][]string, f func(string, string)) {
	for instanceGroup, instances := range instanceCollection {
		for _, instanceIndex := range instances {
			f(instanceGroup, instanceIndex)
		}
	}
}
