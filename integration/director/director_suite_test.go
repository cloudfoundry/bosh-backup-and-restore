package director

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/bosh-backup-and-restore/integration"
	"github.com/cloudfoundry/bosh-backup-and-restore/testcluster"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestDirectorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Director Integration Suite")
}

var pathToPrivateKeyFile string
var pathToPublicKeyFile string
var fixturesDir string

var binary integration.Binary

// knownHostsHome is a per-spec HOME for bbr whose ~/.ssh/known_hosts lists the test director host keys.
var knownHostsHome string

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

var _ = BeforeEach(func() {
	var err error
	knownHostsHome, err = os.MkdirTemp("", "bbr-director-home-")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(os.RemoveAll, knownHostsHome)
	Expect(os.MkdirAll(filepath.Join(knownHostsHome, ".ssh"), 0700)).To(Succeed())
	Expect(os.WriteFile(knownHostsPath(), nil, 0600)).To(Succeed())
})

func knownHostsPath() string {
	return filepath.Join(knownHostsHome, ".ssh", "known_hosts")
}

func knownHostsEnv() string {
	return "HOME=" + knownHostsHome
}

func trustDirector(instance *testcluster.Instance) {
	address := knownhosts.Normalize(instance.Address())
	var lines []string
	for _, key := range instance.HostPublicKeys() {
		publicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(key))
		Expect(err).NotTo(HaveOccurred())
		lines = append(lines, knownhosts.Line([]string{address}, publicKey))
	}
	Expect(lines).NotTo(BeEmpty())

	file, err := os.OpenFile(knownHostsPath(), os.O_APPEND|os.O_WRONLY, 0600)
	Expect(err).NotTo(HaveOccurred())
	_, err = file.WriteString(strings.Join(lines, "\n") + "\n")
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())
}

func readFile(fileName string) string {
	contents, err := os.ReadFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	return string(contents)
}
