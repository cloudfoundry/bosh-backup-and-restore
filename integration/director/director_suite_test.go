package director

import (
	"os"
	"path/filepath"
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

var pathToPrivateKeyFile string
var pathToPublicKeyFile string
var fixturesDir string

var binary integration.Binary

var _ = BeforeSuite(func() {
	commandPath, err := gexec.Build("github.com/cloudfoundry/bosh-backup-and-restore/cmd/bbr")
	Expect(err).NotTo(HaveOccurred())
	binary = integration.NewBinary(commandPath)

	fixturesDir = os.Getenv("FIXTURES_DIR")
	Expect(fixturesDir).NotTo(BeEmpty())

	pathToPrivateKeyFile = filepath.Join(fixturesDir, "test_rsa")
	pathToPublicKeyFile = filepath.Join(fixturesDir, "test_rsa.pub")
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
