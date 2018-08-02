package deployment

import (
	"fmt"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var workspaceDir = "/var/vcap/store/bbr-backup_workspace"
var artifactDir = workspaceDir

var _ = Describe("backup", func() {
	var (
		instanceCollection = map[string][]string{
			"redis":       {"0", "1"},
			"other-redis": {"0"},
		}
		bbrCommand string
	)

	runBBRBackupAndSucceed := func() {
		It("backs up, and cleans up the backup on the remote", func() {
			By("populating data in redis", func() {
				populateRedisFixtureOnInstances(instanceCollection)
			})

			By("running the backup command", func() {
				Eventually(JumpboxInstance.RunCommandAs("vcap", bbrCommand)).Should(gexec.Exit(0))
			})

			By("running the pre-backup lock script", func() {
				runOnInstances(instanceCollection, func(instName, instIndex string) {
					session := RedisDeployment.Instance(instName, instIndex).RunCommand(
						"cat /tmp/pre-backup-lock.out",
					)

					Eventually(session).Should(gexec.Exit(0))
					Expect(session.Out).To(gbytes.Say("output from pre-backup-lock"))
				})
			})

			By("running the post backup unlock script", func() {
				runOnInstances(instanceCollection, func(instName, instIndex string) {
					session := RedisDeployment.Instance(instName, instIndex).RunCommand(
						"cat /tmp/post-backup-unlock.out",
					)
					Eventually(session).Should(gexec.Exit(0))

					Expect(session.Out).To(gbytes.Say("output from post-backup-unlock"))
				})
			})

			By("creating a timestamped directory for holding the artifacts locally", func() {
				session := JumpboxInstance.RunCommandAs("vcap", "ls "+artifactDir)
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say(`\b` + RedisDeployment.Name + `_(\d){8}T(\d){6}Z\b`))
			})

			By("creating the backup artifacts locally", func() {
				JumpboxInstance.AssertFilesExist([]string{
					fmt.Sprintf("%s/%s/redis-0-redis-server.tar", artifactDir, BackupDirWithTimestamp(RedisDeployment.Name)),
					fmt.Sprintf("%s/%s/redis-1-redis-server.tar", artifactDir, BackupDirWithTimestamp(RedisDeployment.Name)),
					fmt.Sprintf("%s/%s/other-redis-0-redis-server.tar", artifactDir, BackupDirWithTimestamp(RedisDeployment.Name)),
				})
			})

			By("cleaning up artifacts from the remote instances", func() {
				runOnInstances(instanceCollection, func(instName, instIndex string) {
					session := RedisDeployment.Instance(instName, instIndex).RunCommand(
						"ls -l /var/vcap/store/bbr-backup",
					)
					Eventually(session).Should(gexec.Exit())
					Expect(session.ExitCode()).To(Equal(1))
					Expect(session.Out).To(gbytes.Say("No such file or directory"))
				})
			})
		})
	}

	Context("when the operator does not specify an artifact directory", func() {
		BeforeEach(func() {
			bbrCommand = fmt.Sprintf(
				`cd %s; BOSH_CLIENT_SECRET=%s ./bbr deployment --ca-cert bosh.crt --username %s --target %s --deployment %s backup`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_ENVIRONMENT"),
				RedisDeployment.Name,
			)
		})

		runBBRBackupAndSucceed()
	})

	Context("when the operator specifies a valid artifact directory", func() {
		BeforeEach(func() {
			artifactDir = workspaceDir + "/artifact-dir"
			Eventually(JumpboxInstance.RunCommandAs("vcap", fmt.Sprintf("mkdir %s", artifactDir))).Should(gexec.Exit(0))

			bbrCommand = fmt.Sprintf(
				`cd %s; BOSH_CLIENT_SECRET=%s ./bbr deployment --ca-cert bosh.crt --username %s --target %s --deployment %s backup --artifact-path %s`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_ENVIRONMENT"),
				RedisDeployment.Name,
				artifactDir,
			)
		})

		runBBRBackupAndSucceed()
	})

	Context("when the operator specifies an artifact directory that does not exist", func() {
		BeforeEach(func() {
			artifactDir = workspaceDir + "/wrong-artifact-dir"

			bbrCommand = fmt.Sprintf(
				`cd %s; BOSH_CLIENT_SECRET=%s ./bbr deployment --ca-cert bosh.crt --username %s --target %s --deployment %s backup --artifact-path %s`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_ENVIRONMENT"),
				RedisDeployment.Name,
				artifactDir,
			)
		})

		It("should fail with an artifact directory does not exist error", func() {
			session := JumpboxInstance.RunCommandAs("vcap", bbrCommand)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say(fmt.Sprintf("%s: no such file or directory", artifactDir)))
		})
	})

	Context("when an instance has many backup scripts", func() {
		It("does not fail draining the artifacts in parallel", func() {
			session := JumpboxInstance.RunCommandAs("vcap", fmt.Sprintf(
				`cd %s; BOSH_CLIENT_SECRET=%s ./bbr deployment --ca-cert bosh.crt --username %s --target %s --deployment %s backup`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_ENVIRONMENT"),
				ManyBbrJobsDeployment.Name,
			))

			Eventually(session).Should(gexec.Exit(0))
		})
	})
})

func populateRedisFixtureOnInstances(instanceCollection map[string][]string) {
	dataFixture := "../../fixtures/redis_test_commands"
	runOnInstances(instanceCollection, func(instName, instIndex string) {
		RedisDeployment.Instance(instName, instIndex).Copy(dataFixture, "/tmp")
		Eventually(
			RedisDeployment.Instance(instName, instIndex).RunCommand(
				"cat /tmp/redis_test_commands | /var/vcap/packages/redis/bin/redis-cli > /dev/null",
			),
		).Should(gexec.Exit(0))
	})
}
