package director

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net"

	"github.com/onsi/gomega/gexec"
	. "github.com/pivotal-cf/bosh-backup-and-restore/system"
)

var _ = Describe("Restores a deployment", func() {
	AfterEach(func() {
		By("removing the backup artifact")
		RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf(
				`sudo rm -rf %s/my-director && sudo rm -f /var/vcap/store/test-backup-and-restore/backup`,
				workspaceDir,
			),
		)
	})

	It("restores", func() {
		By("setting up the jump box")
		Eventually(RunCommandOnRemoteAsVcap(JumpBoxSSHCommand(),
			fmt.Sprintf("sudo mkdir -p %s && sudo chmod -R 0777 %s",
				workspaceDir+"/my-director", workspaceDir))).Should(gexec.Exit(0))
		RunBoshCommand(JumpBoxSCPCommand(), fixturesPath+"bosh-0.tgz", "jumpbox/0:"+workspaceDir+"/my-director")
		RunBoshCommand(JumpBoxSCPCommand(), fixturesPath+"metadata", "jumpbox/0:"+workspaceDir+"/my-director")

		By("running the restore command")
		restoreCommand := RunCommandOnRemote(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s --name my-director restore`,
				workspaceDir,
				MustHaveEnv("HOST_TO_BACKUP"),
			))
		Eventually(restoreCommand).Should(gexec.Exit(0))

		By("ensuring data is restored")
		directorIp, _, _ := net.SplitHostPort(MustHaveEnv("HOST_TO_BACKUP"))
		Eventually(RunCommandOnRemote(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null vcap@%s -i key.pem 'stat /var/vcap/store/test-backup-and-restore/backup'`,
				workspaceDir,
				directorIp,
			),
		)).Should(gexec.Exit(0))
	})
})
