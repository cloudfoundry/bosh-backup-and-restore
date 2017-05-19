package director

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	. "github.com/pivotal-cf/bosh-backup-and-restore/system"
)

var _ = Describe("Backup", func() {
	AfterEach(func() {
		By("removing the backup artifact")
		Eventually(RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf(
				`sudo rm -rf %s/my-director`,
				workspaceDir,
			))).Should(gexec.Exit(0))
	})

	It("backs up the director", func() {
		By("running the backup command")
		backupCommand := RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf(
				`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s --name my-director backup`,
				workspaceDir,
				MustHaveEnv("HOST_TO_BACKUP")),
		)
		Eventually(backupCommand).Should(gexec.Exit(0))

		AssertJumpboxFilesExist([]string{
			fmt.Sprintf("%s/%s/bosh-0.tar", workspaceDir, "my-director"),
		})
	})
})
