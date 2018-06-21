package deployment

import (
	"fmt"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Deployment backup cleanup", func() {
	var deploymentNameToBackup = RedisSlowBackupDeployment.Name

	BeforeEach(func() {
		By("starting a backup and aborting mid-way")
		backupSession := JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(`cd %s; \
			    BOSH_CLIENT_SECRET=%s ./bbr deployment \
			       --ca-cert bosh.crt \
			       --username %s \
			       --target %s \
			       --deployment %s \
			       backup`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_ENVIRONMENT"),
				deploymentNameToBackup),
		)

		Eventually(backupSession.Out).Should(gbytes.Say("Backing up slow-backup on"))
		Eventually(JumpboxInstance.RunCommandAs("vcap", "killall bbr")).Should(gexec.Exit(0))
	})

	Context("When we run backup cleanup", func() {
		It("succeeds", func() {
			By("cleaning up the deployment artifact", func() {
				cleanupCommand := JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(`cd %s; \
						BOSH_CLIENT_SECRET=%s ./bbr deployment \
				    	--ca-cert bosh.crt \
						--username %s \
						--target %s \
						--deployment %s \
						backup-cleanup`,
						workspaceDir,
						MustHaveEnv("BOSH_CLIENT_SECRET"),
						MustHaveEnv("BOSH_CLIENT"),
						MustHaveEnv("BOSH_ENVIRONMENT"),
						deploymentNameToBackup),
				)

				Eventually(cleanupCommand).Should(gexec.Exit(0))
				Expect(cleanupCommand.Out.Contents()).To(ContainSubstring("'%s' cleaned up", deploymentNameToBackup))
			})

			By("allowing subsequent backups to complete successfully", func() {
				backupCommand := JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(`cd %s; \
			    BOSH_CLIENT_SECRET=%s ./bbr deployment \
			       --ca-cert bosh.crt \
			       --username %s \
			       --target %s \
			       --deployment %s \
			       backup`,
						workspaceDir,
						MustHaveEnv("BOSH_CLIENT_SECRET"),
						MustHaveEnv("BOSH_CLIENT"),
						MustHaveEnv("BOSH_ENVIRONMENT"),
						deploymentNameToBackup),
				)
				Eventually(backupCommand).Should(gexec.Exit(0))
			})
		})
	})

	Context("when we don't run a cleanup", func() {
		It("leaves deployment in a state where subsequent backups fail", func() {
			backupCommand := JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(`cd %s; \
			    BOSH_CLIENT_SECRET=%s ./bbr deployment \
			       --ca-cert bosh.crt \
			       --username %s \
			       --target %s \
			       --deployment %s \
			       backup`,
					workspaceDir,
					MustHaveEnv("BOSH_CLIENT_SECRET"),
					MustHaveEnv("BOSH_CLIENT"),
					MustHaveEnv("BOSH_ENVIRONMENT"),
					deploymentNameToBackup),
			)

			Eventually(backupCommand).Should(gexec.Exit(1))
			Expect(backupCommand.Out.Contents()).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance"))

			// clean the corrupted environment for subsequent tests
			Eventually(JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(`cd %s; \
						BOSH_CLIENT_SECRET=%s ./bbr deployment \
						--ca-cert bosh.crt \
						--username %s \
						--target %s \
						--deployment %s \
						backup-cleanup`,
					workspaceDir,
					MustHaveEnv("BOSH_CLIENT_SECRET"),
					MustHaveEnv("BOSH_CLIENT"),
					MustHaveEnv("BOSH_ENVIRONMENT"),
					deploymentNameToBackup),
			)).Should(gexec.Exit(0))
		})
	})
})
