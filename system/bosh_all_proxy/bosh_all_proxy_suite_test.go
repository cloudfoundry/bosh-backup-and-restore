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
	commandPath           string
	err                   error
	ManyScriptsDeployment = DeploymentWithName("many-bbr-jobs")
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(15 * time.Minute)

	By("building bbr")
	commandPath, err = gexec.BuildWithEnvironment("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr", []string{"GOOS=linux", "GOARCH=amd64"})
	Expect(err).NotTo(HaveOccurred())

	By("deploying the many-bbr-jobs deployment")
	ManyScriptsDeployment.Deploy()

})

var _ = AfterSuite(func() {
	By("tearing down the many-bbr-jobs deployment")
	ManyScriptsDeployment.Delete()
})

func TestBoshAllProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BoshAllProxy Suite")
}
