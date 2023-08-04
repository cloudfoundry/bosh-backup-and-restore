package deployment

import (
	"fmt"
	"time"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo/v2"
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
	commandPath    string
	boshCaCertPath string
	err            error
)

var fixturesPath = "../../fixtures/redis-backup/"

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(15 * time.Minute)

	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer GinkgoRecover()
		defer wg.Done()

		By("deploying the Redis test release")
		RedisDeployment.Deploy()

		By("deploying the Redis with backup_one_restore_all property")
		RedisWithBackupOneRestoreAll.Deploy()

		By("deploying the Redis with disabled bbr job property")
		RedisDeploymentWithDisabledJob.Deploy()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()

		By("deploying the slow backup Redis test release")
		RedisSlowBackupDeployment.Deploy()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()

		By("deploying the Redis with locking order release")
		RedisWithLockingOrderDeployment.Deploy()

		By("deploying the jump box")
		JumpboxDeployment.Deploy()

		By("deploying the many-bbr-jobs deployment")
		ManyBbrJobsDeployment.Deploy()
	}()

	wg.Wait()

	By("building bbr")
	commandPath, err = gexec.BuildWithEnvironment("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr", []string{"GOOS=linux", "GOARCH=amd64"})
	Expect(err).NotTo(HaveOccurred())

	By("setting up the jump box")
	Eventually(JumpboxInstance.RunCommand(
		fmt.Sprintf("sudo mkdir %s && sudo chown vcap:vcap %s && sudo chmod 0777 %s", workspaceDir, workspaceDir, workspaceDir))).Should(gexec.Exit(0))

	By("writing $BOSH_CA_CERT to a temp file")
	boshCaCertPath, err = WriteEnvVarToTempFile("BOSH_CA_CERT")
	Expect(err).NotTo(HaveOccurred())

	By("copying bbr and bosh.crt to the jumpbox")
	JumpboxInstance.Copy(commandPath, workspaceDir+"/bbr")
	JumpboxInstance.Copy(boshCaCertPath, workspaceDir+"/bosh.crt")
})

var _ = AfterSuite(func() {
	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer GinkgoRecover()
		defer wg.Done()

		By("tearing down the redis release")
		RedisDeployment.Delete()

		By("tearing down the Redis with backup_one_restore_all property")
		RedisWithBackupOneRestoreAll.Delete()

		By("tearing down the Redis with disabled bbr job property")
		RedisDeploymentWithDisabledJob.Delete()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()

		By("tearing down the slow backup Redis test release")
		RedisSlowBackupDeployment.Delete()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()

		By("tearing down the Redis with locking order release")
		RedisWithLockingOrderDeployment.Delete()

		By("tearing down the jump box")
		JumpboxDeployment.Delete()

		By("tearing down the many-bbr-jobs deployment")
		ManyBbrJobsDeployment.Delete()
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
