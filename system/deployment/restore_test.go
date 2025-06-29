package deployment

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restores a deployment", func() {
	It("restores", func() {
		var (
			workspaceDir       = "/var/vcap/store/restore_workspace"
			backupMetadata     = "../../fixtures/redis-backup/metadata"
			instanceCollection = map[string][]string{
				"redis":       {"0", "1"},
				"other-redis": {"0"},
			}
			backupName = "redis-backup_20170502T132536Z"
		)

		By("setting up the jump box")
		Eventually(JumpboxInstance.RunCommand(
			fmt.Sprintf("sudo mkdir -p %s && sudo chown -R vcap:vcap %s && sudo chmod -R 0777 %s",
				workspaceDir+"/"+backupName, workspaceDir, workspaceDir),
		)).Should(gexec.Exit(0))

		JumpboxInstance.Copy(commandPath, workspaceDir+"/bbr")
		JumpboxInstance.Copy(boshCaCertPath, workspaceDir+"/bosh.crt")
		JumpboxInstance.Copy(backupMetadata, workspaceDir+"/"+backupName+"/metadata")
		runOnInstances(instanceCollection, func(in, ii string) {
			fileName := fmt.Sprintf("%s-%s-redis-server.tar", in, ii)
			JumpboxInstance.Copy(
				fixturesPath+fileName,
				fmt.Sprintf("%s/%s/%s", workspaceDir, backupName, fileName),
			)
		})

		By("running the restore command")
		Eventually(JumpboxInstance.RunCommand(
			fmt.Sprintf(`cd %s;
				BOSH_CLIENT_SECRET=%s ./bbr \
				deployment --debug \
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
				RedisDeployment.Name,
				backupName,
			),
		)).Should(gexec.Exit(0))

		By("running the pre-restore-lock script")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			session := RedisDeployment.Instance(instName, instIndex).RunCommand(
				"cat /tmp/pre-restore-lock.out",
			)

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("output from pre-restore-lock"))
		})

		By("running the post-restore-unlock script")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			session := RedisDeployment.Instance(instName, instIndex).RunCommand(
				"cat /tmp/post-restore-unlock.out",
			)

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("output from post-restore-unlock"))
		})

		By("cleaning up artifacts from the remote instances")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			session := RedisDeployment.Instance(instName, instIndex).RunCommand(
				"ls -l /var/vcap/store/bbr-backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})

		By("ensuring data is restored")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			Eventually(RedisDeployment.Instance(instName, instIndex).RunCommand(
				fmt.Sprintf("sudo ls -la /var/vcap/store/redis-server"), //nolint:staticcheck
			)).Should(gexec.Exit(0))

			redisSession := RedisDeployment.Instance(instName, instIndex).RunCommand(
				"/var/vcap/packages/redis/bin/redis-cli -a redis get FOO23",
			)

			Eventually(redisSession.Out).Should(gbytes.Say("BAR23"))
		})
	})

	Context("when a job is disabled", func() {
		It("restores only the enabled job", func() {
			var (
				workspaceDir   = "/var/vcap/store/restore_workspace"
				backupMetadata = "../../fixtures/redis-backup-with-disabled-job/metadata"

				backupName = "redis-backup_20170502T132536Z"
			)

			By("setting up the jump box")
			Eventually(JumpboxInstance.RunCommand(
				fmt.Sprintf("sudo mkdir -p %s && sudo chown -R vcap:vcap %s && sudo chmod -R 0777 %s",
					workspaceDir+"/"+backupName, workspaceDir, workspaceDir),
			)).Should(gexec.Exit(0))

			JumpboxInstance.Copy(commandPath, workspaceDir)
			JumpboxInstance.Copy(boshCaCertPath, workspaceDir+"/bosh.crt")
			JumpboxInstance.Copy(backupMetadata, workspaceDir+"/"+backupName+"/metadata")
			fileName := "redis-0-redis-server.tar"
			JumpboxInstance.Copy(
				fixturesPath+fileName,
				fmt.Sprintf("%s/%s/%s", workspaceDir, backupName, fileName),
			)

			bbrSession := JumpboxInstance.RunCommand(
				fmt.Sprintf(`cd %s;
				BOSH_CLIENT_SECRET=%s ./bbr \
				deployment --debug \
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
					RedisDeploymentWithDisabledJob.Name,
					backupName,
				),
			)
			By("running the restore command")
			Eventually(bbrSession).Should(gexec.Exit(0))

			By("running the restore scripts for the non-disabled job", func() {
				session := RedisDeploymentWithDisabledJob.Instance("redis", "0").RunCommand(
					"cat /tmp/pre-restore-lock.out",
				)

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("output from pre-restore-lock"))

				session = RedisDeploymentWithDisabledJob.Instance("redis", "0").RunCommand(
					"cat /tmp/post-restore-unlock.out",
				)

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("output from post-restore-unlock"))
			})

			By("not running the restore scripts for the disabled job", func() {
				session := RedisDeploymentWithDisabledJob.Instance("disabled-job", "0").RunCommand(
					"cat /tmp/pre-restore-lock.out",
				)

				Eventually(session).Should(gexec.Exit())
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(string(session.Out.Contents())).To(ContainSubstring("No such file"))

				session = RedisDeploymentWithDisabledJob.Instance("disabled-job", "0").RunCommand(
					"cat /tmp/restore.out",
				)

				Eventually(session).Should(gexec.Exit())
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(string(session.Out.Contents())).To(ContainSubstring("No such file"))

				session = RedisDeploymentWithDisabledJob.Instance("disabled-job", "0").RunCommand(
					"cat /tmp/post-restore-unlock.out",
				)

				Eventually(session).Should(gexec.Exit())
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(string(session.Out.Contents())).To(ContainSubstring("No such file"))

				Expect(string(bbrSession.Buffer().Contents())).To(MatchRegexp(`Found disabled jobs on instance disabled-job\/.* jobs: disabled-job`))

			})

			By("cleaning up artifacts from the remote instances")
			session := RedisDeploymentWithDisabledJob.Instance("redis", "0").RunCommand(
				"ls -l /var/vcap/store/bbr-backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))

			By("ensuring data is restored")
			Eventually(RedisDeploymentWithDisabledJob.Instance("redis", "0").RunCommand(
				fmt.Sprintf("sudo ls -la /var/vcap/store/redis-server"), //nolint:staticcheck
			)).Should(gexec.Exit(0))

			redisSession := RedisDeploymentWithDisabledJob.Instance("redis", "0").RunCommand(
				"/var/vcap/packages/redis/bin/redis-cli -a redis get FOO23",
			)

			Eventually(redisSession.Out).Should(gbytes.Say("BAR23"))
		})
	})
})
