package system

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("backup with custom metadata", func() {
	var instanceCollection = map[string][]string{
		"redis-server-with-metadata": {"0"},
	}

	It("backs up, gives the backup artifact the desired custom name and cleans up", func() {
		By("populating data in redis")
		populateRedisWithMetadata(instanceCollection )

		By("running the backup command")
		Eventually(RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf(
				`cd %s; BOSH_CLIENT_SECRET=%s ./pbr --ca-cert bosh.crt --username %s --target %s --deployment %s backup`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_URL"),
				RedisWithMetadataDeployment()),
		),
		).Should(gexec.Exit(0))

		By("creating the named backup artifacts locally")
		AssertJumpboxFilesExist([]string{
			fmt.Sprintf("%s/%s/custom-redis-backup.tgz", workspaceDir, RedisWithMetadataDeployment()),
		})

		By("cleaning up artifacts from the remote instances")
		runOnAllInstances(instanceCollection, func(instName, instIndex string) {
			session := RunCommandOnRemote(RedisWithMetadataDeploymentSSHCommand(instName, instIndex),
				"ls -l /var/vcap/store/backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})
	})
})

func populateRedisWithMetadata(instanceCollection map[string][]string) {
	dataFixture := "../fixtures/redis_test_commands"
	runOnAllInstances(instanceCollection, func(instName, instIndex string) {
		RunBoshCommand(RedisWithMetadataDeploymentSCPCommand(), dataFixture, fmt.Sprintf("%s/%s:/tmp", instName, instIndex))
		Eventually(
			RunCommandOnRemote(RedisWithMetadataDeploymentSSHCommand(instName, instIndex),
				"cat /tmp/redis_test_commands | /var/vcap/packages/redis/bin/redis-cli > /dev/null",
			),
		).Should(gexec.Exit(0))
	})
}
