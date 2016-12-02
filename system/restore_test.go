package system

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restores a deployment", func() {
	var workspaceDir = "/var/vcap/store/restore_workspace"
	var backupMetadata = "../fixtures/redis-backup/metadata"

	It("restores", func() {
		By("setting up the jump box")
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(), fmt.Sprintf("sudo mkdir -p %s && sudo chown -R vcap:vcap %s && sudo chmod -R 0777 %s",
				workspaceDir+"/"+RedisDeployment(), workspaceDir, workspaceDir),
		)).Should(gexec.Exit(0))

		RunBoshCommand(JumpBoxSCPCommand(), MustHaveEnv("BOSH_CERT_PATH"), "jumpbox/0:"+workspaceDir+"/bosh.crt")
		RunBoshCommand(JumpBoxSCPCommand(), commandPath, "jumpbox/0:"+workspaceDir)
		RunBoshCommand(JumpBoxSCPCommand(), backupMetadata, "jumpbox/0:"+workspaceDir+"/"+RedisDeployment()+"/metadata")
		performOnAllInstances(func(in, ii string) {
			fileName := fmt.Sprintf("%s-%s.tgz", in, ii)
			RunBoshCommand(
				JumpBoxSCPCommand(),
				fixturesPath+fileName,
				fmt.Sprintf(
					"jumpbox/0:%s/%s/%s",
					workspaceDir,
					RedisDeployment(),
					fileName,
				),
			)
		})

		By("running the restore command")
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; BOSH_PASSWORD=%s ./pbr --debug --ca-cert bosh.crt --username %s --target %s --deployment %s restore`,
				workspaceDir, MustHaveEnv("BOSH_PASSWORD"), MustHaveEnv("BOSH_USER"), MustHaveEnv("BOSH_URL"), RedisDeployment()),
		)).Should(gexec.Exit(0))

		performOnAllInstances(func(instName, instIndex string) {
			Eventually(RunCommandOnRemote(
				RedisDeploymentSSHCommand(instName, instIndex),
				fmt.Sprintf("sudo ls -la /var/vcap/store/redis-server"),
			)).Should(gexec.Exit(0))

			redisSession := RunCommandOnRemote(RedisDeploymentSSHCommand(instName, instIndex),
				"/var/vcap/packages/redis/bin/redis-cli -a redis get FOO23",
			)

			Eventually(redisSession.Out).Should(gbytes.Say("BAR23"))
		})
	})
})
