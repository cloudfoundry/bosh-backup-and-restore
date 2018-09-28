package all_deployments_tests

import (
	"fmt"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
)

var _ = Describe("All deployments", func() {
	Context("pre-backup-check", func() {
		It("Can run pre-backup-check on all deployments", func() {
			cmd := exec.Command(
				commandPath,
				"deployment",
				"--ca-cert", MustHaveEnv("BOSH_CA_CERT"),
				"--username", MustHaveEnv("BOSH_CLIENT"),
				"--password", MustHaveEnv("BOSH_CLIENT_SECRET"),
				"--target", MustHaveEnv("BOSH_ENVIRONMENT"),
				"--all-deployments",
				"pre-backup-check",
			)
			cmd.Dir = tempDirPath
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Expect(session.Out).To(gbytes.Say("Deployment 'redis-1' can be backed up."))
			Expect(session.Out).To(gbytes.Say("Deployment 'redis-2' can be backed up."))
			Expect(session.Out).To(gbytes.Say("Deployment 'redis-3' can be backed up."))
			Expect(session.Out).To(gbytes.Say("All 3 deployments can be backed up"))
		})
	})

	Context("backup", func() {
		var redisInstance1 Instance
		var redisInstance2 Instance
		var redisInstance3 Instance
		var redisDeployment1 Deployment
		var redisDeployment2 Deployment
		var redisDeployment3 Deployment

		BeforeEach(func() {
			redisDeployment1 = NewDeployment("redis-1", "")
			redisInstance1 = redisDeployment1.Instance("redis", "0")

			redisDeployment2 = NewDeployment("redis-2", "")
			redisInstance2 = redisDeployment2.Instance("redis", "0")

			redisDeployment3 = NewDeployment("redis-3", "")
			redisInstance3 = redisDeployment3.Instance("redis", "0")

		})

		AfterEach(func() {
			cleanupLockScriptOutput(redisInstance1)
			cleanupLockScriptOutput(redisInstance2)
			cleanupLockScriptOutput(redisInstance3)
		})

		It("Can run backup on all deployments", func() {
			cmd := exec.Command(
				commandPath,
				"deployment",
				"--ca-cert", MustHaveEnv("BOSH_CA_CERT"),
				"--username", MustHaveEnv("BOSH_CLIENT"),
				"--password", MustHaveEnv("BOSH_CLIENT_SECRET"),
				"--target", MustHaveEnv("BOSH_ENVIRONMENT"),
				"--all-deployments",
				"backup",
			)
			cmd.Dir = tempDirPath
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			By("providing debug output", func() {
				Expect(session.Out).To(gbytes.Say("Starting backup of redis-1"))
				Expect(session.Out).To(gbytes.Say("Backup created of redis-1"))
				Expect(session.Out).To(gbytes.Say("Starting backup of redis-2"))
				Expect(session.Out).To(gbytes.Say("Backup created of redis-2"))
				Expect(session.Out).To(gbytes.Say("Starting backup of redis-3"))
				Expect(session.Out).To(gbytes.Say("Backup created of redis-3"))
				Expect(session.Out).To(gbytes.Say("All 3 deployments backed up."))
			})

			By("running the pre-backup lock script", func() {
				AssertPreBackupLock(redisInstance1)
				AssertPreBackupLock(redisInstance2)
				AssertPreBackupLock(redisInstance3)
			})

			By("running the post backup unlock script", func() {
				AssertPostBackupUnlock(redisInstance1)
				AssertPostBackupUnlock(redisInstance2)
				AssertPostBackupUnlock(redisInstance3)
			})

			By("creating a timestamped directory for holding the artifacts locally", func() {
				AssertTimestampedDirectoryCreated(redisDeployment1)
				AssertTimestampedDirectoryCreated(redisDeployment2)
				AssertTimestampedDirectoryCreated(redisDeployment3)
			})

			By("creating the backup artifacts locally", func() {
				AssertBackupArtifactsCreated(redisDeployment1)
				AssertBackupArtifactsCreated(redisDeployment2)
				AssertBackupArtifactsCreated(redisDeployment3)
			})

			By("cleaning up artifacts from the remote instances", func() {
				AssertArtifactsRemovedFromInstance(redisInstance1)
				AssertArtifactsRemovedFromInstance(redisInstance2)
				AssertArtifactsRemovedFromInstance(redisInstance3)
			})
		})

	})

	Context("backup-cleanup", func() {
		var redisInstance1 Instance
		var redisInstance2 Instance
		var redisInstance3 Instance

		BeforeEach(func() {
			redisInstance1 = NewDeployment("redis-1", "").Instance("redis", "0")
			redisInstance2 = NewDeployment("redis-2", "").Instance("redis", "0")
			redisInstance3 = NewDeployment("redis-3", "").Instance("redis", "0")

			session := redisInstance3.RunCommand("sudo mkdir /var/vcap/store/bbr-backup")
			Eventually(session).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			cleanupLockScriptOutput(redisInstance1)
			cleanupLockScriptOutput(redisInstance2)
			cleanupLockScriptOutput(redisInstance3)
		})

		It("Can run backup-cleanup on all deployments", func() {
			cmd := exec.Command(
				commandPath,
				"deployment",
				"--ca-cert", MustHaveEnv("BOSH_CA_CERT"),
				"--username", MustHaveEnv("BOSH_CLIENT"),
				"--password", MustHaveEnv("BOSH_CLIENT_SECRET"),
				"--target", MustHaveEnv("BOSH_ENVIRONMENT"),
				"--all-deployments",
				"backup-cleanup",
			)

			cmd.Dir = tempDirPath
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			By("providing debug output", func() {
				//Expect(session.Out).To(gbytes.Say("Starting backup of redis-1"))
				//Expect(session.Out).To(gbytes.Say("Backup created of redis-1"))
				//Expect(session.Out).To(gbytes.Say("Starting backup of redis-2"))
				//Expect(session.Out).To(gbytes.Say("Backup created of redis-2"))
				//Expect(session.Out).To(gbytes.Say("Starting backup of redis-3"))
				//Expect(session.Out).To(gbytes.Say("Backup created of redis-3"))
				//Expect(session.Out).To(gbytes.Say("All 3 deployments backed up."))
			})

			By("running the post backup unlock script", func() {
				AssertPostBackupUnlock(redisInstance1)
				AssertPostBackupUnlock(redisInstance2)
				AssertPostBackupUnlock(redisInstance3)
			})

			By("cleaning up artifacts from the remote instances", func() {
				AssertArtifactsRemovedFromInstance(redisInstance3)
			})
		})
	})
})

func cleanupLockScriptOutput(instance Instance) {
	session := instance.RunCommand("sudo rm /tmp/post-backup-unlock.out")
	Eventually(session).Should(gexec.Exit())

	session = instance.RunCommand("sudo rm /tmp/pre-backup-lock.out")
	Eventually(session).Should(gexec.Exit())

}

func AssertArtifactsRemovedFromInstance(instance Instance) {
	session := instance.RunCommand(
		"ls -l /var/vcap/store/bbr-backup",
	)
	Eventually(session).Should(gexec.Exit())
	Expect(session.ExitCode()).To(Equal(1))
	Expect(session.Out).To(gbytes.Say("No such file or directory"))
}

func AssertTimestampedDirectoryCreated(deployment Deployment) {
	cmd := exec.Command("ls", ".")
	cmd.Dir = tempDirPath
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
	Expect(session.Out).To(gbytes.Say(`\b` + deployment.Name + `_(\d){8}T(\d){6}Z\b`))
}

func AssertBackupArtifactsCreated(deployment Deployment) {
	files, err := filepath.Glob(filepath.Join(tempDirPath, fmt.Sprintf("%s/redis-0-redis-server.tar", BackupDirWithTimestamp(deployment.Name))))
	Expect(err).NotTo(HaveOccurred())
	Expect(files).To(HaveLen(1))
}

func AssertPostBackupUnlock(instance Instance) {
	session := instance.RunCommand(
		"cat /tmp/post-backup-unlock.out",
	)
	Eventually(session).Should(gexec.Exit(0))
	Expect(session.Out).To(gbytes.Say("output from post-backup-unlock"))
}

func AssertPreBackupLock(instance Instance) {
	session := instance.RunCommand(
		"cat /tmp/pre-backup-lock.out",
	)
	Eventually(session).Should(gexec.Exit(0))
	Expect(session.Out).To(gbytes.Say("output from pre-backup-lock"))
}
