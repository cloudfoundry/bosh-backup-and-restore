package deployment

import (
	"fmt"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("lifecycle with backup_one_restore_all enabled", func() {
	var instances = map[string][]string{
		"redis": {"0", "1"},
	}

	It("backs up, gives the backup artifact the desired custom name and cleans up", func() {
		By("populating data in redis")
		redisSession := RedisWithBackupOneRestoreAll.Instance("redis", "0").RunCommand(
			"/var/vcap/packages/redis/bin/redis-cli -a redis set FOO1 BAR1",
		)
		Eventually(redisSession).ShouldNot(gexec.Exit(0))

		By("running the backup command")
		Eventually(JumpboxInstance.RunCommandAs("vcap",
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
				MustHaveEnv("BOSH_ENVIRONMENT"),
				RedisWithBackupOneRestoreAll.Name),
		),
		).Should(gexec.Exit(0))

		By("creating the named backup artifacts locally")
		JumpboxInstance.AssertFilesExist([]string{
			fmt.Sprintf("%s/%s/redis-server-redis-test-backup-one-restore-all.tar", workspaceDir, BackupDirWithTimestamp(RedisWithBackupOneRestoreAll.Name)),
			fmt.Sprintf("%s/%s/redis-1-redis-server.tar", workspaceDir, BackupDirWithTimestamp(RedisWithBackupOneRestoreAll.Name)),
		})

		By("cleaning up artifacts from the remote instances")
		runOnInstances(instances, func(instName, instIndex string) {
			session := RedisWithBackupOneRestoreAll.Instance(instName, instIndex).RunCommand(
				"ls -l /var/vcap/store/bbr-backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})

		By("removing the data")
		redisSession = RedisWithBackupOneRestoreAll.Instance("redis", "0").RunCommand(
			"/var/vcap/packages/redis/bin/redis-cli -a redis del FOO1",
		)
		Eventually(redisSession).Should(gexec.Exit(0))

		By("asserting none of the instances have the data")
		redisSession = RedisWithBackupOneRestoreAll.Instance("redis", "0").RunCommand(
			"/var/vcap/packages/redis/bin/redis-cli -a redis get FOO1",
		)
		Eventually(redisSession).ShouldNot(gexec.Exit(0))

		redisSession = RedisWithBackupOneRestoreAll.Instance("redis", "1").RunCommand(
			"/var/vcap/packages/redis/bin/redis-cli -a redis get FOO1",
		)
		Eventually(redisSession).ShouldNot(gexec.Exit(0))

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
				MustHaveEnv("BOSH_ENVIRONMENT"),
				RedisWithBackupOneRestoreAll.Name,
				BackupDirWithTimestamp(RedisWithBackupOneRestoreAll.Name)),
		)).Should(gexec.Exit(0))

		By("cleaning up artifacts from the remote restored instances")
		runOnInstances(instances, func(instName, instIndex string) {
			session := RedisWithBackupOneRestoreAll.Instance(instName, instIndex).RunCommand(
				"ls -l /var/vcap/store/bbr-backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})

		By("ensuring data is restored in both instances")
		runOnInstances(instances, func(instName, instIndex string) {
			Eventually(RedisWithBackupOneRestoreAll.Instance(instName, instIndex).RunCommand(
				fmt.Sprintf("sudo ls -la /var/vcap/store/redis-server"),
			)).Should(gexec.Exit(0))

			redisSession := RedisWithBackupOneRestoreAll.Instance(instName, instIndex).RunCommand(
				"/var/vcap/packages/redis/bin/redis-cli -a redis get FOO1",
			)

			Eventually(redisSession.Out).Should(gbytes.Say("BAR1"))
		})
	})
})
