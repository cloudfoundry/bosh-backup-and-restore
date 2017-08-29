package deployment

import (
	"fmt"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("backup with specified locking order", func() {
	var redisInstance = map[string][]string{
		"redis": {"0"},
	}
	var allInstances = map[string][]string{
		"redis":           {"0"},
		"capi":            {"0"},
		"capi-consumer-1": {"0"},
		"capi-consumer-2": {"0"},
	}

	It("locks the instances in the correct order and backs up", func() {
		By("populating the Redis instance")
		populateRedis(redisInstance)

		By("running the backup command")
		backupSession := JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(`cd %s;
			BOSH_CLIENT_SECRET=%s ./bbr \
			deployment \
			--ca-cert bosh.crt \
			--username %s \
			--target %s \
			--deployment %s \
			backup`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_URL"),
				RedisWithLockingOrderDeployment.Name),
		)
		Eventually(backupSession).Should(gexec.Exit(0))

		By("locking the instances in the correct order", func() {
			Eventually(backupSession.Out).Should(gbytes.Say("Locking capi-consumer-[12]"))
			Eventually(backupSession.Out).Should(gbytes.Say("Locking capi-consumer-[12]"))
			Eventually(backupSession.Out).Should(gbytes.Say("Locking capi"))
			Eventually(backupSession.Out).Should(gbytes.Say("Locking redis"))
		})

		By("cleaning up artifacts from the remote instances")
		runOnInstances(allInstances, func(instName, instIndex string) {
			session := RedisWithLockingOrderDeployment.Instance(instName, instIndex).RunCommand(
				"ls -l /var/vcap/store/bbr-backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})

		By("unlocking the instances in the correct order", func() {
			Eventually(backupSession.Out).Should(gbytes.Say("Unlocking redis"))
			Eventually(backupSession.Out).Should(gbytes.Say("Unlocking capi"))
			Eventually(backupSession.Out).Should(gbytes.Say("Unlocking capi-consumer-[12]"))
			Eventually(backupSession.Out).Should(gbytes.Say("Unlocking capi-consumer-[12]"))
		})

		By("running the restore command")
		Eventually(JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(`cd %s;
			BOSH_CLIENT_SECRET=%s ./bbr \
			deployment \
			--debug \
			--ca-cert bosh.crt \
			--username %s \
			--target %s \
			--deployment %s \
			restore \
			--artifact-path %s`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_URL"),
				RedisWithLockingOrderDeployment.Name,
				BackupDirWithTimestamp(RedisWithLockingOrderDeployment.Name)),
		)).Should(gexec.Exit(0))

		By("cleaning up artifacts from the remote restored instances")
		runOnInstances(allInstances, func(instName, instIndex string) {
			session := RedisWithLockingOrderDeployment.Instance(instName, instIndex).RunCommand(
				"ls -l /var/vcap/store/bbr-backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})

		By("ensuring data is restored")
		runOnInstances(redisInstance, func(instName, instIndex string) {
			Eventually(RedisWithLockingOrderDeployment.Instance(instName, instIndex).RunCommand(
				fmt.Sprintf("sudo ls -la /var/vcap/store/redis-server"),
			)).Should(gexec.Exit(0))

			redisSession := RedisWithLockingOrderDeployment.Instance(instName, instIndex).RunCommand(
				"/var/vcap/packages/redis/bin/redis-cli -a redis get FOO23",
			)

			Eventually(redisSession.Out).Should(gbytes.Say("BAR23"))
		})
	})
})

func populateRedis(instanceCollection map[string][]string) {
	dataFixture := "../../fixtures/redis_test_commands"
	runOnInstances(instanceCollection, func(instName, instIndex string) {
		RedisWithLockingOrderDeployment.Instance(instName, instIndex).Copy(dataFixture, "/tmp")
		Eventually(
			RedisWithLockingOrderDeployment.Instance(instName, instIndex).RunCommand(
				"cat /tmp/redis_test_commands | /var/vcap/packages/redis/bin/redis-cli > /dev/null",
			),
		).Should(gexec.Exit(0))
	})
}
