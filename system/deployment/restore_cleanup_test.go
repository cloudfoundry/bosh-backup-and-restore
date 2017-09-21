package deployment

import (
	"fmt"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Deployment restore cleanup", func() {
	var deploymentNameToRestore = RedisSlowBackupDeployment.Name
	var backupArtifactName = "redis-with-slow-backup"
	var backupArtifactPath = "../../fixtures/" + backupArtifactName + ".tar"
	var workspaceDir = "/var/vcap/store/restore_cleanup_workspace"

	BeforeEach(func() {
		By("copying the backup artifact on the jumpbox", func() {
			Eventually(JumpboxInstance.RunCommand(
				fmt.Sprintf("sudo mkdir -p %s && sudo chown -R vcap:vcap %s && sudo chmod -R 0777 %s",
					workspaceDir, workspaceDir, workspaceDir),
			)).Should(gexec.Exit(0))

			JumpboxInstance.Copy(MustHaveEnv("BOSH_CERT_PATH"), workspaceDir+"/bosh.crt")
			JumpboxInstance.Copy(commandPath, workspaceDir)
			JumpboxInstance.Copy(backupArtifactPath, workspaceDir)
			Eventually(JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf("cd %s; tar xvf redis-with-slow-backup.tar", workspaceDir),
			)).Should(gexec.Exit(0))
		})

		By("starting a restore and aborting mid-way")
		restoreSession := JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(`cd %s; \
			    BOSH_CLIENT_SECRET=%s ./bbr deployment \
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
				deploymentNameToRestore,
				backupArtifactName),
		)

		Eventually(restoreSession.Out).Should(gbytes.Say("Restoring slow-backup"))
		Eventually(JumpboxInstance.RunCommandAs("vcap", "killall bbr")).Should(gexec.Exit(0))
	})

	AfterEach(func() {
		By("cleaning up the deployment", func() {
			Eventually(JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(`cd %s; \
						BOSH_CLIENT_SECRET=%s ./bbr deployment \
						--ca-cert bosh.crt \
						--username %s \
						--target %s \
						--deployment %s \
						restore-cleanup`,
					workspaceDir,
					MustHaveEnv("BOSH_CLIENT_SECRET"),
					MustHaveEnv("BOSH_CLIENT"),
					MustHaveEnv("BOSH_URL"),
					deploymentNameToRestore),
			)).Should(gexec.Exit(0))
		})
	})

	Context("when we run restore cleanup", func() {
		It("succeeds", func() {
			By("cleaning up the deployment artifact", func() {
				cleanupCommand := JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(`cd %s; \
						BOSH_CLIENT_SECRET=%s ./bbr deployment \
						--ca-cert bosh.crt \
						--username %s \
						--target %s \
						--deployment %s \
						restore-cleanup`,
						workspaceDir,
						MustHaveEnv("BOSH_CLIENT_SECRET"),
						MustHaveEnv("BOSH_CLIENT"),
						MustHaveEnv("BOSH_URL"),
						deploymentNameToRestore),
				)

				Eventually(cleanupCommand).Should(gexec.Exit(0))
				Expect(cleanupCommand.Out.Contents()).To(ContainSubstring("'%s' cleaned up", deploymentNameToRestore))
			})

			By("allowing subsequent restores to complete successfully", func() {
				restoreCommand := JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(`cd %s; \
						BOSH_CLIENT_SECRET=%s ./bbr deployment \
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
						deploymentNameToRestore,
						backupArtifactName),
				)

				Eventually(restoreCommand).Should(gexec.Exit(0))
			})
		})
	})

	Context("when we don't run a cleanup", func() {
		It("is in a state where subsequent restores fail", func() {
			restoreCommand := JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(`cd %s; \
					BOSH_CLIENT_SECRET=%s ./bbr deployment \
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
					deploymentNameToRestore,
					backupArtifactName),
			)

			Eventually(restoreCommand).Should(gexec.Exit(1))
			Expect(restoreCommand.Out.Contents()).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance"))
		})
	})
})
