package deployment

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
				MustHaveEnv("BOSH_URL"),
				deploymentNameToBackup),
		)

		Eventually(backupSession.Out).Should(gbytes.Say("Backing up slow-backup on"))
		time.Sleep(5 * time.Second)
		Eventually(backupSession.Kill()).Should(gexec.Exit())
	})

	It("succeeds", func() {
		By("cleaning up the deployment", func() {
			cleanupCommand := JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(`cd %s; \
			    BOSH_CLIENT_SECRET=%s ./bbr deployment \
			       --ca-cert bosh.crt \
			       --username %s \
			       --target %s \
			       --deployment %s \
			       cleanup`,
					workspaceDir,
					MustHaveEnv("BOSH_CLIENT_SECRET"),
					MustHaveEnv("BOSH_CLIENT"),
					MustHaveEnv("BOSH_URL"),
					deploymentNameToBackup),
			)

			Eventually(cleanupCommand).Should(gexec.Exit(0))
			Expect(cleanupCommand.Out.Contents()).To(ContainSubstring("'%s' cleaned up", deploymentNameToBackup))
		})

		By("backup completing successfully", func() {
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
					MustHaveEnv("BOSH_URL"),
					deploymentNameToBackup),
			)

			Eventually(backupCommand).Should(gexec.Exit(0))
		})
	})
})
