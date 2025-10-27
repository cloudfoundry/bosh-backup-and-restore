package director

import (
	"os"
	"testing"

	"github.com/cloudfoundry/bosh-backup-and-restore/integration"
	"github.com/cloudfoundry/bosh-backup-and-restore/testcluster"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestDirectorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Director Integration Suite")
}

var pathToPrivateKeyFile = "../../../fixtures/test_rsa"
var pathToPublicKeyFile = "../../fixtures/test_rsa.pub"

var binary integration.Binary

var _ = BeforeSuite(func() {
	commandPath, err := gexec.Build("github.com/cloudfoundry/bosh-backup-and-restore/cmd/bbr")
	Expect(err).NotTo(HaveOccurred())
	binary = integration.NewBinary(commandPath)
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	testcluster.WaitForContainersToDie()
})

func readFile(fileName string) string {
	contents, err := os.ReadFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	return string(contents)
}
