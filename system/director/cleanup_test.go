package director

import (
	"fmt"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/pivotal-cf/bosh-backup-and-restore/system"
)

var _ = Describe("cleanup", func() {
	var directorIP = MustHaveEnv("HOST_TO_BACKUP")

	BeforeEach(func() {
		By("starting a backup and aborting mid-way")
		backupSession := JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(
				`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s backup`,
				workspaceDir,
				directorIP),
		)

		Eventually(backupSession.Out).Should(gbytes.Say("Backing up test-backup-and-restore on bosh"))
		time.Sleep(5 * time.Second)
		Eventually(backupSession.Kill()).Should(gexec.Exit())
	})

	It("succeeds", func() {
		By("cleaning up the deployment", func() {
			cleanupCommand := JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(
					`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s cleanup`,
					workspaceDir,
					directorIP),
			)

			Eventually(cleanupCommand).Should(gexec.Exit(0))
			Eventually(cleanupCommand).Should(gbytes.Say("'%s' cleaned up", directorIP))
		})

		time.Sleep(5 * time.Second) //TODO: why is this necessary?

		By("backup completing successfully", func() {
			backupCommand := JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(
					`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s backup`,
					workspaceDir,
					directorIP),
			)

			Eventually(backupCommand).Should(gexec.Exit(0))
		})
	})
})
