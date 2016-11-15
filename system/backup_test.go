package system

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Backs up a deployment", func() {
	var commandPath string
	var err error
	var workspaceDir = "/var/vcap/store/backup_workspace"

	BeforeSuite(func() {
		SetDefaultEventuallyTimeout(60 * time.Second)
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
	It("backs up", func() {
		By("polulating data in redis")
		dataFixture := "../fixtures/redis_test_commands"
		RunBoshCommand(TestDeploymentSCPCommand(), dataFixture, "redis/0:/tmp")
		Eventually(
			RunCommandOnRemote(TestDeploymentSSHCommand(),
				"cat /tmp/redis_test_data | /var/vcap/packages/redis/bin/redis-cli",
			)).Should(gexec.Exit(0))

		By("setting up the jump box")
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(), fmt.Sprintf("sudo mkdir %s && sudo chown vcap:vcap %s && sudo chmod 0777 %s", workspaceDir, workspaceDir, workspaceDir),
		)).Should(gexec.Exit(0))
		RunBoshCommand(JumpBoxSCPCommand(), commandPath, "jumpbox/0:"+workspaceDir)
		RunBoshCommand(JumpBoxSCPCommand(), MustHaveEnv("BOSH_CERT_PATH"), "jumpbox/0:"+workspaceDir+"/bosh.crt")

		By("running the backup command")
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; BOSH_PASSWORD=%s ./pbr --ca-cert bosh.crt --username %s --target %s --deployment %s backup`,
				workspaceDir, MustHaveEnv("BOSH_PASSWORD"), MustHaveEnv("BOSH_USER"), MustHaveEnv("BOSH_URL"), TestDeployment()),
		)).Should(gexec.Exit(0))

		By("checking backup artifact has been created")
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(), fmt.Sprintf("ls %s/%s", workspaceDir, TestDeployment()),
		)).Should(gbytes.Say("redis-0.tgz"))

	})
	AfterSuite(func() {
		RunBoshCommand(TestDeploymentBoshCommand(), "delete-deployment")
		RunBoshCommand(JumpBoxBoshCommand(), "delete-deployment")
	})
})
