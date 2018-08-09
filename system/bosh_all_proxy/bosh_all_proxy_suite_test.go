package bosh_all_proxy

import (
	"testing"

	"time"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	commandPath              string
	err                      error
	ManyBbrScriptsDeployment = DeploymentWithName("many-bbr-scripts")
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(15 * time.Minute)

	By("building bbr")
	commandPath, err = gexec.BuildWithEnvironment("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr", []string{"GOOS=linux", "GOARCH=amd64"})
	Expect(err).NotTo(HaveOccurred())

})

func TestBoshAllProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BoshAllProxy Suite")
}
