package system

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Backs up a deployment", func() {
	var workspaceDir = "/var/vcap/store/backup_workspace"

	It("backs up", func() {
		By("polulating data in redis")
		dataFixture := "../fixtures/redis_test_commands"
		RunBoshCommand(TestDeploymentSCPCommand(), dataFixture, "redis/0:/tmp")
		Eventually(
			RunCommandOnRemote(TestDeploymentSSHCommand(),
				"cat /tmp/redis_test_commands | /var/vcap/packages/redis/bin/redis-cli > /dev/null",
			)).Should(gexec.Exit(0))

		By("setting up the jump box")
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(), fmt.Sprintf("sudo mkdir %s && sudo chown vcap:vcap %s && sudo chmod 0777 %s", workspaceDir, workspaceDir, workspaceDir),
		)).Should(gexec.Exit(0))
		RunBoshCommand(JumpBoxSCPCommand(), commandPath, "jumpbox/0:"+workspaceDir)
		RunBoshCommand(JumpBoxSCPCommand(), MustHaveEnv("BOSH_CERT_PATH"), "jumpbox/0:"+workspaceDir+"/bosh.crt")

		By("running the backup command")
		Eventually(RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; BOSH_PASSWORD=%s ./pbr --ca-cert bosh.crt --username %s --target %s --deployment %s backup`,
				workspaceDir, MustHaveEnv("BOSH_PASSWORD"), MustHaveEnv("BOSH_USER"), MustHaveEnv("BOSH_URL"), TestDeployment()),
		)).Should(gexec.Exit(0))

		By("checking backup artifact has been created")
		Eventually(RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(), fmt.Sprintf("ls %s/%s", workspaceDir, TestDeployment()),
		)).Should(gbytes.Say("redis-0.tgz"))
	})
})
