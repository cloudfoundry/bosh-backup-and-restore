package system

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "System Suite")
}

var (
	commandPath string
	err         error
)

var _ = BeforeEach(func() {
	SetDefaultEventuallyTimeout(2 * time.Minute)
	// TODO: tests should build and upload the test release
	// By("Creating the test release")
	// RunBoshCommand(testDeploymentBoshCommand, "create-release", "--dir=../fixtures/releases/redis-test-release/", "--force")
	// By("Uploading the test release")
	// RunBoshCommand(testDeploymentBoshCommand, "upload-release", "--dir=../fixtures/releases/redis-test-release/", "--rebase")

	By("deploying the test release")
	RunBoshCommand(TestDeploymentBoshCommand(), "deploy", TestDeploymentManifest())
	By("deploying the jump box")
	RunBoshCommand(JumpBoxBoshCommand(), "deploy", JumpboxDeploymentManifest())

	commandPath, err = gexec.BuildWithEnvironment("github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr", []string{"GOOS=linux", "GOARCH=amd64"})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterEach(func() {
	RunBoshCommand(TestDeploymentBoshCommand(), "delete-deployment")
	RunBoshCommand(JumpBoxBoshCommand(), "delete-deployment")
})
