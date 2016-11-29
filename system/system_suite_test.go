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

	wg.Add(2)
	go func() {
		By("deploying the test release")
		RunBoshCommand(RedisDeploymentBoshCommand(), "deploy", "--var=deployment-name="+fmt.Sprintf("redis-%s", TestEnv()), RedisDeploymentManifest())
		wg.Done()
	}()

	go func() {
		By("deploying the jump box")
		RunBoshCommand(JumpBoxBoshCommand(), "deploy", "--var=deployment-name="+fmt.Sprintf("jumpbox-%s", TestEnv()), JumpboxDeploymentManifest())
		wg.Done()
	}()
	wg.Wait()

	commandPath, err = gexec.BuildWithEnvironment("github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr", []string{"GOOS=linux", "GOARCH=amd64"})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterEach(func() {
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		By("tearing down the test release")
		RunBoshCommand(RedisDeploymentBoshCommand(), "delete-deployment")
		wg.Done()
	}()

	go func() {
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
