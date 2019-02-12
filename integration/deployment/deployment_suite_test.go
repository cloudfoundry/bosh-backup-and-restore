package deployment

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"

	"io/ioutil"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/integration"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
)

func TestDeploymentIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployment Integration Suite")
}

//Default cert for golang ssh
var sslCertPath = "../../../fixtures/test.crt"
var sslCertValue string

var binary integration.Binary

const bbrVersion = "bbr_version"

var _ = BeforeSuite(func() {
	commandPath, err := gexec.Build("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr", "-ldflags", fmt.Sprintf("-X main.version=%s", bbrVersion))
	Expect(err).NotTo(HaveOccurred())
	binary = integration.NewBinary(commandPath)

	contents, err := ioutil.ReadFile("../../fixtures/test.crt")
	Expect(err).NotTo(HaveOccurred())
	sslCertValue = string(contents)
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	testcluster.WaitForContainersToDie()
})

func newIndex(index int) *int {
	return &index
}
