package system

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Multiple deployments ", func() {
	XIt("backs up", func() {
		By("running the backup command")
		Eventually(RunCommandOnRemoteAsVcap(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`cd %s; BOSH_PASSWORD=%s ./pbr --ca-cert bosh.crt --username %s --target %s --deployment %s,%s backup`,
				workspaceDir,
				MustHaveEnv("BOSH_PASSWORD"),
				MustHaveEnv("BOSH_USER"),
				MustHaveEnv("BOSH_URL"),
				RedisDeployment(),
				AnotherRedisDeployment(),
			),
		)).Should(gexec.Exit(0))

		By("checking backup artifact has been created for deployment 1")
		AssertJumpboxFilesExist([]string{
			fmt.Sprintf("%s/%s/redis-0.tgz", workspaceDir, RedisDeployment()),
			fmt.Sprintf("%s/%s/redis-1.tgz", workspaceDir, RedisDeployment()),
			fmt.Sprintf("%s/%s/other-redis-0.tgz", workspaceDir, RedisDeployment()),
		})

		By("checking backup artifact has been created for deployment 2")
		AssertJumpboxFilesExist([]string{
			fmt.Sprintf("%s/%s/another-redis-0.tgz", workspaceDir, AnotherRedisDeployment()),
		})
	})
})
