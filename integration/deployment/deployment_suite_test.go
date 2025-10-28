package deployment

import (
	"encoding/pem"
	"fmt"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"

	"github.com/cloudfoundry/bosh-backup-and-restore/integration"
	"github.com/cloudfoundry/bosh-backup-and-restore/testcluster"
)

func TestDeploymentIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployment Integration Suite")
}

// Default cert for golang ssh
var sslCertPath string
var sslCertValue string

var binary integration.Binary

const bbrVersion = "bbr_version"

var _ = BeforeSuite(func() {
	commandPath, err := gexec.Build("github.com/cloudfoundry/bosh-backup-and-restore/cmd/bbr", "-ldflags", fmt.Sprintf("-X main.version=%s", bbrVersion))
	Expect(err).NotTo(HaveOccurred())
	binary = integration.NewBinary(commandPath)

	x509Cert := httptest.NewTLSServer(nil).Certificate()
	pem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: x509Cert.Raw,
	})
	sslCertValue = string(pem)

	sslCertFile, err := os.CreateTemp("", "golang-httptest-certificate-")
	Expect(err).NotTo(HaveOccurred())
	sslCertPath = sslCertFile.Name()
	_, err = sslCertFile.Write(pem)
	Expect(err).NotTo(HaveOccurred())

})

var _ = AfterSuite(func() {
	os.Remove(sslCertPath) //nolint:errcheck
	gexec.CleanupBuildArtifacts()
	testcluster.WaitForContainersToDie()
})

func newIndex(index int) *int {
	return &index
}
