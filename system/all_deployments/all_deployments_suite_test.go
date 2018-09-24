package all_deployments_tests

import (
	"testing"
	"time"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	commandPath string
	err         error
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(15 * time.Minute)

	By("building bbr")
	commandPath, err = gexec.Build("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr")
	Expect(err).NotTo(HaveOccurred())

})

func TestBoshTeam(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "All Deployments Suite")
}
