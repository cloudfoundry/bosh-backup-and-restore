package deployment

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/pivotal-cf/bosh-backup-and-restore/system"
)

var workspaceDir = "/var/vcap/store/backup_workspace"

var _ = Describe("backup", func() {
	var instanceCollection = map[string][]string{
		"redis":       {"0", "1"},
		"other-redis": {"0"},
	}

	It("backs up, and cleans up the backup on the remote", func() {
		By("populating data in redis")
		populateRedisFixtureOnInstances(instanceCollection)

		By("running the backup command")
		Eventually(RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; BOSH_CLIENT_SECRET=%s ./bbr deployment --ca-cert bosh.crt --username %s --target %s --deployment %s backup`,
				workspaceDir, MustHaveEnv("BOSH_CLIENT_SECRET"), MustHaveEnv("BOSH_CLIENT"), MustHaveEnv("BOSH_URL"), RedisDeployment()),
		)).Should(gexec.Exit(0))

		By("running the pre-backup lock script")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			session := RunCommandOnRemote(RedisDeploymentSSHCommand(instName, instIndex),
				"cat /tmp/pre-backup-lock.out",
			)

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).Should(ContainSubstring("output from pre-backup-lock"))
		})

		By("running the post backup unlock script")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			session := RunCommandOnRemote(RedisDeploymentSSHCommand(instName, instIndex),
				"cat /tmp/post-backup-unlock.out",
			)
			Eventually(session).Should(gexec.Exit(0))

			Expect(session.Out.Contents()).Should(ContainSubstring("output from post-backup-unlock"))
		})

		By("creating the backup artifacts locally")
		AssertJumpboxFilesExist([]string{
			fmt.Sprintf("%s/%s/redis-0.tar", workspaceDir, RedisDeployment()),
			fmt.Sprintf("%s/%s/redis-1.tar", workspaceDir, RedisDeployment()),
			fmt.Sprintf("%s/%s/other-redis-0.tar", workspaceDir, RedisDeployment()),
		})

		By("cleaning up artifacts from the remote instances")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			session := RunCommandOnRemote(RedisDeploymentSSHCommand(instName, instIndex),
				"ls -l /var/vcap/store/backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})
	})
})

func populateRedisFixtureOnInstances(instanceCollection map[string][]string) {
	dataFixture := "../../fixtures/redis_test_commands"
	runOnInstances(instanceCollection, func(instName, instIndex string) {
		RunBoshCommand(RedisDeploymentSCPCommand(), dataFixture, fmt.Sprintf("%s/%s:/tmp", instName, instIndex))
		Eventually(
			RunCommandOnRemote(RedisDeploymentSSHCommand(instName, instIndex),
				"cat /tmp/redis_test_commands | /var/vcap/packages/redis/bin/redis-cli > /dev/null",
			),
		).Should(gexec.Exit(0))
	})
}
