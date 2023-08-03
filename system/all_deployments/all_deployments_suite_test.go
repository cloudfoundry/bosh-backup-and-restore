package all_deployments_tests

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo/v2"
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

	workingDir, err = ioutil.TempDir("/tmp", "workingDir")

})

func TestBoshTeam(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "All Deployments Suite")
}
