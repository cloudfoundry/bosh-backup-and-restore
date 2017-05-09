package deployment

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/pivotal-cf/bosh-backup-and-restore/system"

	"sync"
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

var fixturesPath = "../../fixtures/redis-backup/"

var _ = BeforeEach(func() {
	SetDefaultEventuallyTimeout(4 * time.Minute)
	//By("Creating the test release")
	//RunBoshCommand(GenericBoshCommand(), "create-release", "--dir=../../fixtures/releases/redis-test-release/", "--force")
	//By("Uploading the test release")
	//RunBoshCommand(GenericBoshCommand(), "upload-release", "--dir=../../fixtures/releases/redis-test-release/", "--rebase")
	var wg sync.WaitGroup

	wg.Add(4)
	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the Redis test release")
		RunBoshCommand(RedisDeploymentBoshCommand(), "deploy", SetName(RedisDeployment()), RedisDeploymentManifest())
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the Redis with metadata")
		RunBoshCommand(RedisWithMetadataDeploymentBoshCommand(), "deploy", SetName(RedisWithMetadataDeployment()), RedisWithMetadataDeploymentManifest())
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the jump box")
		RunBoshCommand(JumpBoxBoshCommand(), "deploy", SetName(JumpboxDeployment()), JumpboxDeploymentManifest())
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the other Redis test release")
		RunBoshCommand(AnotherRedisDeploymentBoshCommand(), "deploy", SetName(AnotherRedisDeployment()), AnotherRedisDeploymentManifest())
	}()
	wg.Wait()

	By("building bbr")
	commandPath, err = gexec.BuildWithEnvironment("github.com/pivotal-cf/bosh-backup-and-restore/cmd/bbr", []string{"GOOS=linux", "GOARCH=amd64"})
	Expect(err).NotTo(HaveOccurred())

	By("setting up the jump box")
	Eventually(RunCommandOnRemote(
		JumpBoxSSHCommand(), fmt.Sprintf("sudo mkdir %s && sudo chown vcap:vcap %s && sudo chmod 0777 %s", workspaceDir, workspaceDir, workspaceDir),
	)).Should(gexec.Exit(0))
	RunBoshCommand(JumpBoxSCPCommand(), commandPath, "jumpbox/0:"+workspaceDir)
	RunBoshCommand(JumpBoxSCPCommand(), MustHaveEnv("BOSH_CERT_PATH"), "jumpbox/0:"+workspaceDir+"/bosh.crt")
})

var _ = AfterEach(func() {
	var wg sync.WaitGroup

	wg.Add(4)
	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the redis release")
		RunBoshCommand(RedisDeploymentBoshCommand(), "delete-deployment")
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the other redis release")
		RunBoshCommand(RedisWithMetadataDeploymentBoshCommand(), "delete-deployment")
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the redis with metadata")
		RunBoshCommand(AnotherRedisDeploymentBoshCommand(), "delete-deployment")
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the jump box")
		RunBoshCommand(JumpBoxBoshCommand(), "delete-deployment")
	}()

	wg.Wait()
})

func runOnInstances(instanceCollection map[string][]string, f func(string, string)) {
	for instanceGroup, instances := range instanceCollection {
		for _, instanceIndex := range instances {
			f(instanceGroup, instanceIndex)
		}
	}
}

func SetName(name string) string {
	return "--var=deployment-name=" + name
}
