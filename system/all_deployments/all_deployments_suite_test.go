package all_deployments_tests

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	commandPath  string
	err          error
	artifactPath string
	workingDir   string
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(15 * time.Minute)

	By("building bbr")
	commandPath, err = gexec.Build("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr")
	Expect(err).NotTo(HaveOccurred())

	artifactPath, err = ioutil.TempDir("/tmp", "all_deployments")

	workingDir, err = ioutil.TempDir("/tmp", "workingDir")

})

var _ = AfterSuite(func() {
	os.RemoveAll(artifactPath)
})

func TestBoshTeam(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "All Deployments Suite")
}
