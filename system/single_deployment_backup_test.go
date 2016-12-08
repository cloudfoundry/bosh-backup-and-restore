package system

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var workspaceDir = "/var/vcap/store/backup_workspace"

var _ = Describe("Single deployment", func() {
	It("backs up, and cleans up the backup on the remote", func() {
		By("populating data in redis")
		dataFixture := "../fixtures/redis_test_commands"

		RunBoshCommand(RedisDeploymentSCPCommand(), dataFixture, "redis/0:/tmp")

		performOnAllInstances(func(instName, instIndex string) {
			Eventually(
				RunCommandOnRemote(RedisDeploymentSSHCommand(instName, instIndex),
					"cat /tmp/redis_test_commands | /var/vcap/packages/redis/bin/redis-cli > /dev/null",
				),
			).Should(gexec.Exit(0))
		})

		By("running the backup command")
		Eventually(RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; BOSH_PASSWORD=%s ./pbr --ca-cert bosh.crt --username %s --target %s --deployment %s backup`,
				workspaceDir, MustHaveEnv("BOSH_PASSWORD"), MustHaveEnv("BOSH_USER"), MustHaveEnv("BOSH_URL"), RedisDeployment()),
		)).Should(gexec.Exit(0))

		By("creating the backup artifacts locally")
		AssertJumpboxFilesExist([]string{
			fmt.Sprintf("%s/%s/redis-0.tgz", workspaceDir, RedisDeployment()),
			fmt.Sprintf("%s/%s/redis-1.tgz", workspaceDir, RedisDeployment()),
			fmt.Sprintf("%s/%s/other-redis-0.tgz", workspaceDir, RedisDeployment()),
		})

		By("cleaning up artifacts from the remote instances")
		performOnAllInstances(func(instName, instIndex string) {
			session := RunCommandOnRemote(RedisDeploymentSSHCommand(instName, instIndex),
				"ls -l /var/vcap/store/backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})
	})
})
