package system

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

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

var instanceCollection = map[string][]string{
	"redis":       []string{"0", "1"},
	"other-redis": []string{"0"},
}
var fixturesPath = "../fixtures/redis-backup/"

var _ = BeforeEach(func() {
	SetDefaultEventuallyTimeout(2 * time.Minute)
	// TODO: tests should build and upload the test release
	// By("Creating the test release")
	// RunBoshCommand(testDeploymentBoshCommand, "create-release", "--dir=../fixtures/releases/redis-test-release/", "--force")
	// By("Uploading the test release")
	// RunBoshCommand(testDeploymentBoshCommand, "upload-release", "--dir=../fixtures/releases/redis-test-release/", "--rebase")
	var wg sync.WaitGroup

	wg.Add(3)
	go func() {
		GinkgoRecover()
		By("deploying the Redis test release")
		RunBoshCommand(RedisDeploymentBoshCommand(), "deploy", SetName(RedisDeployment()), RedisDeploymentManifest())
		wg.Done()
	}()

	go func() {
		GinkgoRecover()
		By("deploying the jump box")
		RunBoshCommand(JumpBoxBoshCommand(), "deploy", SetName(JumpboxDeployment()), JumpboxDeploymentManifest())
		wg.Done()
	}()

	go func() {
		GinkgoRecover()
		By("deploying the other Redis test release")
		RunBoshCommand(AnotherRedisDeploymentBoshCommand(), "deploy", SetName(AnotherRedisDeployment()), AnotherRedisDeploymentManifest())
		wg.Done()
	}()
	wg.Wait()

	By("building pbr")
	commandPath, err = gexec.BuildWithEnvironment("github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr", []string{"GOOS=linux", "GOARCH=amd64"})
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

	wg.Add(3)
	go func() {
		GinkgoRecover()
		By("tearing down the redis release")
		RunBoshCommand(RedisDeploymentBoshCommand(), "delete-deployment")
		wg.Done()
	}()
	go func() {
		GinkgoRecover()
		By("tearing down the other redis release")
		RunBoshCommand(AnotherRedisDeploymentBoshCommand(), "delete-deployment")
		wg.Done()
	}()

	go func() {
		GinkgoRecover()
		By("tearing down the jump box")
		RunBoshCommand(JumpBoxBoshCommand(), "delete-deployment")
		wg.Done()
	}()

	wg.Wait()
})

func performOnAllInstances(f func(string, string)) {
	for instanceGroup, instances := range instanceCollection {
		for _, instanceIndex := range instances {
			f(instanceGroup, instanceIndex)
		}
	}
}

func SetName(name string) string {
	return "--var=deployment-name=" + name
}
