package deployment

import (
	"fmt"
	"os"
	"os/exec"

	. "github.com/cloudfoundry/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo/v2"
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

	boshAllProxy := fmt.Sprintf(
		"ssh+socks5://%s@%s?private-key=%s",
		MustHaveEnv("BOSH_GW_USER"),
		MustHaveEnv("BOSH_GW_HOST"),
		MustHaveEnv("BOSH_GW_PRIVATE_KEY"),
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

	Context("BOSH_ALL_PROXY is set", func() {
		It("backs up the deployment using BOSH_ALL_PROXY", func() {
			cmd := exec.Command(
				commandPath,
				"deployment",
				"--ca-cert", MustHaveEnv("BOSH_CA_CERT"),
				"--username", MustHaveEnv("BOSH_CLIENT"),
				"--password", MustHaveEnv("BOSH_CLIENT_SECRET"),
				"--target", MustHaveEnv("BOSH_ENVIRONMENT"),
				"--deployment", RedisDeployment.Name,
				"backup",
			)
			cmd.Env = append(os.Environ(), "BOSH_ALL_PROXY="+boshAllProxy)
			cmd.Stderr = GinkgoWriter
			cmd.Stdout = GinkgoWriter

			fmt.Println("BOSH_ALL_PROXY=", boshAllProxy, " bbr ", cmd.Args)

			Expect(cmd.Run()).To(Succeed())
		})
	})

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

			Eventually(session, "30m").Should(gexec.Exit(0))
		})
	})

	Context("when an instance has a backup job that is disabled", func() {
		It("does not run the scripts", func() {
			By("running a backup")
			bbrCommand = fmt.Sprintf(
				`cd %s; BOSH_CLIENT_SECRET=%s ./bbr deployment --ca-cert bosh.crt --username %s --debug --target %s --deployment %s backup`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_ENVIRONMENT"),
				RedisDeploymentWithDisabledJob.Name,
			)
			session := JumpboxInstance.RunCommandAs("vcap", bbrCommand)
			Eventually(session).Should(gexec.Exit(0))

			By("calling the scripts of the non-disabled jobs", func() {
				session := RedisDeploymentWithDisabledJob.Instance("redis", "0").RunCommand(
					"cat /tmp/pre-backup-lock.out",
				)

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("output from pre-backup-lock"))

				session = RedisDeploymentWithDisabledJob.Instance("redis", "0").RunCommand(
					"cat /tmp/post-backup-unlock.out",
				)

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("output from post-backup-unlock"))

			})

			By("not calling the scripts of the disabled jobs", func() {
				session := RedisDeploymentWithDisabledJob.Instance("disabled-job", "0").RunCommand(
					"cat /tmp/pre-backup-lock.out",
				)

				Eventually(session).Should(gexec.Exit())
				Expect(session.ExitCode()).NotTo(Equal(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring("No such file"))

				session = RedisDeploymentWithDisabledJob.Instance("disabled-job", "0").RunCommand(
					"cat /tmp/backup.out",
				)

				Eventually(session).Should(gexec.Exit())
				Expect(session.ExitCode()).NotTo(Equal(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring("No such file"))

				session = RedisDeploymentWithDisabledJob.Instance("disabled-job", "0").RunCommand(
					"cat /tmp/post-backup-unlock.out",
				)

				Eventually(session).Should(gexec.Exit())
				Expect(session.ExitCode()).NotTo(Equal(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring("No such file"))
			})

			By("logging", func() {
				Expect(string(session.Buffer().Contents())).To(MatchRegexp(`Found disabled jobs on instance disabled-job\/.* jobs: disabled-job`))
			})
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
