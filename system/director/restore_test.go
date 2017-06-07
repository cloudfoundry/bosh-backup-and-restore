package director

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
	. "github.com/pivotal-cf/bosh-backup-and-restore/system"
)

var _ = Describe("Restores a deployment", func() {
	var restorePath = "/var/vcap/store/test-backup-and-restore"
	var restoredArtifactPath = restorePath + "/backup"
	var skipSSHFingerprintCheckOpts = "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
	var artifactName = "artifactToRestore"

	AfterEach(func() {
		directorIP := MustHaveEnv("HOST_TO_BACKUP")

		By("cleaning up the jump box")
		Eventually(RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf(
				`sudo rm -rf %s/%s`,
				workspaceDir,
				artifactName,
			),
		)).Should(gexec.Exit(0))

		By("cleaning up the director")
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; ssh %s vcap@%s -i key.pem 'sudo rm -rf %s'`,
				workspaceDir,
				skipSSHFingerprintCheckOpts,
				directorIP,
				restorePath,
			),
		)).Should(gexec.Exit(0))
	})

	It("restores", func() {
		directorIP := MustHaveEnv("HOST_TO_BACKUP")

		By("setting up the jump box")
		Eventually(RunCommandOnRemoteAsVcap(JumpBoxSSHCommand(),
			fmt.Sprintf("sudo mkdir -p %s && sudo chmod -R 0777 %s",
				workspaceDir+"/"+artifactName, workspaceDir))).Should(gexec.Exit(0))
		RunBoshCommand(JumpBoxSCPCommand(), fixturesPath+"bosh-0-amazing-backup-and-restore.tar", "jumpbox/0:"+workspaceDir+"/"+artifactName)
		RunBoshCommand(JumpBoxSCPCommand(), fixturesPath+"bosh-0-remarkable-backup-and-restore.tar", "jumpbox/0:"+workspaceDir+"/"+artifactName)
		RunBoshCommand(JumpBoxSCPCommand(), fixturesPath+"bosh-0-test-backup-and-restore.tar", "jumpbox/0:"+workspaceDir+"/"+artifactName)
		RunBoshCommand(JumpBoxSCPCommand(), fixturesPath+"metadata", "jumpbox/0:"+workspaceDir+"/"+artifactName)

		By("running the restore command")
		restoreCommand := RunCommandOnRemote(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s restore --artifact-path %s`,
				workspaceDir,
				directorIP,
				artifactName,
			))
		Eventually(restoreCommand).Should(gexec.Exit(0))

		By("ensuring data is restored")
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; ssh %s vcap@%s -i key.pem 'stat %s'`,
				workspaceDir,
				skipSSHFingerprintCheckOpts,
				directorIP,
				restoredArtifactPath,
			),
		)).Should(gexec.Exit(0))
	})
})
