package windows

import (
	"fmt"

	. "github.com/cloudfoundry/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("pre-backup-check", func() {
	Context("when the deployment includes a windows VM", func() {
		It("confirms the deployment can be backed up", func() {
			preBackupCheckCommand := JumpboxWindowsInstance.RunCommandAs("vcap",
				fmt.Sprintf(`cd %s; \
					BOSH_CLIENT_SECRET=%s ./bbr deployment \
					--ca-cert bosh.crt \
					--username %s \
					--target %s \
					--deployment %s \
					pre-backup-check`,
					workspaceDir,
					MustHaveEnv("BOSH_CLIENT_SECRET"),
					MustHaveEnv("BOSH_CLIENT"),
					MustHaveEnv("BOSH_ENVIRONMENT"),
					RedisWindowsDeployment.Name,
				),
			)

			Eventually(preBackupCheckCommand).Should(gexec.Exit(0))

			backupableMessage := fmt.Sprintf("Deployment '%s' can be backed up", RedisWindowsDeployment.Name)
			Expect(preBackupCheckCommand.Out.Contents()).To(ContainSubstring(backupableMessage))

			skippingMessage := "skipping Windows instance windows-vm/"
			Expect(preBackupCheckCommand.Out.Contents()).To(ContainSubstring(skippingMessage))
		})
	})
})
