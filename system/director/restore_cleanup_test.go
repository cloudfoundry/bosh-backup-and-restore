package director

import (
	"fmt"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Director restore cleanup", func() {
	var directorIP = MustHaveEnv("HOST_TO_BACKUP")
	var artifactName = "artifactToRestore"
	var workspaceDir = "/var/vcap/store/restore_cleanup_workspace"

	BeforeEach(func() {
		By("setting up the jumpbox", func() {
			Eventually(JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf("sudo mkdir -p %s && sudo chown -R vcap:vcap %s && sudo chmod -R 0777 %s",
					workspaceDir+"/"+artifactName, workspaceDir, workspaceDir),
			)).Should(gexec.Exit(0))
			JumpboxInstance.Copy(fixturesPath+"bosh-0-amazing-backup-and-restore.tar", workspaceDir+"/"+artifactName)
			JumpboxInstance.Copy(fixturesPath+"bosh-0-remarkable-backup-and-restore.tar", workspaceDir+"/"+artifactName)
			JumpboxInstance.Copy(fixturesPath+"bosh-0-test-backup-and-restore.tar", workspaceDir+"/"+artifactName)
			JumpboxInstance.Copy(fixturesPath+"metadata", workspaceDir+"/"+artifactName)

			JumpboxInstance.Copy(MustHaveEnv("SSH_KEY"), workspaceDir+"/key.pem")
			Eventually(JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf("sudo chmod 400 %s", workspaceDir+"/key.pem"),
			)).Should(gexec.Exit(0))
			JumpboxInstance.Copy(commandPath, workspaceDir)
		})

		By("starting a restore and aborting mid-way", func() {
			restoreSession := JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(`cd %s; \
			./bbr director \
			--username vcap \
			--private-key-path ./key.pem \
			--host %s restore \
			--artifact-path %s`,
					workspaceDir,
					directorIP,
					artifactName,
				))
			Eventually(restoreSession.Out).Should(gbytes.Say("Restoring test-backup-and-restore on bosh"))
			Eventually(JumpboxInstance.RunCommandAs("vcap", "killall bbr")).Should(gexec.Exit(0))
		})
	})

	AfterEach(func() {
		By("cleaning up the director", func() {
			Eventually(JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(
					`cd %s; \
					ssh %s vcap@%s \
						-i key.pem \
						"sudo rm -rf /var/vcap/store/bbr-backup"`,
					workspaceDir,
					skipSSHFingerprintCheckOpts,
					directorIP,
				))).Should(gexec.Exit(0))
		})

		By("removing the backup", func() {
			Eventually(JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(
					`sudo rm -rf %s/%s*`,
					workspaceDir,
					directorIP,
				))).Should(gexec.Exit(0))
		})
	})

	Context("When we run cleanup", func() {
		It("succeeds", func() {
			By("cleaning up the director artifact", func() {
				cleanupCommand := JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(
						`cd %s; \
					 ./bbr director \
						 --username vcap \
						 --debug \
						 --private-key-path ./key.pem \
						 --host %s restore-cleanup`,
						workspaceDir,
						directorIP),
				)

				Eventually(cleanupCommand).Should(gexec.Exit(0))
				Eventually(cleanupCommand).Should(gbytes.Say("'%s' cleaned up", directorIP))

				Eventually(JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(
						`cd %s; \
						ssh %s vcap@%s \
						-i key.pem \
						"ls -l /var/vcap/store/bbr-backup"`,
						workspaceDir,
						skipSSHFingerprintCheckOpts,
						directorIP,
					))).Should(gbytes.Say("ls: cannot access /var/vcap/store/bbr-backup: No such file or directory"))
			})

			By("allowing subsequent restore to complete successfully", func() {
				restoreCommand := JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(
						`cd %s; \
					 ./bbr director \
						 --debug \
						 --username vcap \
						 --private-key-path ./key.pem \
						 --host %s restore \
						 --artifact-path %s`,
						workspaceDir,
						directorIP,
						artifactName),
				)

				Eventually(restoreCommand).Should(gexec.Exit(0))
			})
		})
	})

	Context("when we don't run a cleanup", func() {
		It("is in a state where subsequent restore fail", func() {
			restoreCommand := JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(
					`cd %s; \
					 ./bbr director \
						 --username vcap \
						 --private-key-path ./key.pem \
						 --host %s restore \
						 --artifact-path %s`,
					workspaceDir,
					directorIP,
					artifactName),
			)

			Eventually(restoreCommand).Should(gexec.Exit(1))
			Expect(restoreCommand.Out.Contents()).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists"))
		})
	})
})
