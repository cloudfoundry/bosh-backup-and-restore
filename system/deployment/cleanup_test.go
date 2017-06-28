package deployment

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/pivotal-cf/bosh-backup-and-restore/system"
)

var _ = Describe("cleanup", func() {
	XIt("backs up, and cleans up the backup on the remote", func() {
		cleanupCommand := JumpboxDeployment().Instance("jumpbox", "0").RunCommandAs("vcap",
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
				MustHaveEnv("BOSH_URL"),
				RedisDeployment().Name),
		)

		Eventually(cleanupCommand).Should(gexec.Exit(0))
		Expect(cleanupCommand.Out.Contents()).To(ContainSubstring("Deployment 'redis-dev-1' can be backed up"))
	})
})
