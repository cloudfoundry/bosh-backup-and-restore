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
	var instanceCollection = map[string][]string{
		"redis":       {"0", "1"},
		"other-redis": {"0"},
	}

	It("restores", func() {
		By("setting up the jump box")
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(), fmt.Sprintf("sudo mkdir -p %s && sudo chown -R vcap:vcap %s && sudo chmod -R 0777 %s",
				workspaceDir+"/"+RedisDeployment(), workspaceDir, workspaceDir),
		)).Should(gexec.Exit(0))

		RunBoshCommand(JumpBoxSCPCommand(), MustHaveEnv("BOSH_CERT_PATH"), "jumpbox/0:"+workspaceDir+"/bosh.crt")
		RunBoshCommand(JumpBoxSCPCommand(), commandPath, "jumpbox/0:"+workspaceDir)
		RunBoshCommand(JumpBoxSCPCommand(), backupMetadata, "jumpbox/0:"+workspaceDir+"/"+RedisDeployment()+"/metadata")
		runOnInstances(instanceCollection, func(in, ii string) {
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
			fmt.Sprintf(`cd %s; BOSH_CLIENT_SECRET=%s ./bbr --debug --ca-cert bosh.crt --username %s --target %s --deployment %s restore`,
				workspaceDir, MustHaveEnv("BOSH_CLIENT_SECRET"), MustHaveEnv("BOSH_CLIENT"), MustHaveEnv("BOSH_URL"), RedisDeployment()),
		)).Should(gexec.Exit(0))

		By("cleaning up artifacts from the remote instances")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			session := RunCommandOnRemote(RedisDeploymentSSHCommand(instName, instIndex),
				"ls -l /var/vcap/store/backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})

		By("ensuring data is restored")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
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
