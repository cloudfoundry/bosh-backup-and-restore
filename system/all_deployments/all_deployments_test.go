package all_deployments_tests

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
)

var _ = Describe("All deployments", func() {
	var redis1 = "redis-1"
	var redis2 = "redis-2"
	var redis3 = "redis-3"

	Context("when running pre-backup-check", func() {
		Context("and all deployments are backupable", func() {
			It("reports that all deployments are backupable", func() {
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
				cmd.Dir = artifactPath
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				output := strings.Split(string(session.Out.Contents()), "\n")
				output[1] = strings.TrimSpace(output[1])
				output[2] = strings.TrimSpace(output[2])
				output[3] = strings.TrimSpace(output[3])

				Expect(output[0]).To(ContainSubstring("Pending: redis-1, redis-2, redis-3"))
				Expect(output[1]).To(ContainSubstring("-------------------------"))
				Expect(output[2:5]).To(ConsistOf(
					ContainSubstring("Deployment 'redis-1' can be backed up."),
					ContainSubstring("Deployment 'redis-2' can be backed up."),
					ContainSubstring("Deployment 'redis-3' can be backed up."),
				))
				Expect(output[5]).To(ContainSubstring("-------------------------"))
				Expect(output[6]).To(ContainSubstring("Successfully can be backed up: redis-1, redis-2, redis-3"))
				Expect(output[7]).To(ContainSubstring(""))
				Expect(output).To(HaveLen(8))
			})
		})

		Context("and some deployments are not backupable", func() {
			BeforeEach(func() {
				moveBackupScript("redis-1", "/var/vcap/jobs/redis-server", "/tmp/redis-server")
				moveBackupScript("redis-2", "/var/vcap/jobs/redis-server", "/tmp/redis-server")
			})

			AfterEach(func() {
				moveBackupScript("redis-1", "/tmp/redis-server", "/var/vcap/jobs/redis-server")
				moveBackupScript("redis-2", "/tmp/redis-server", "/var/vcap/jobs/redis-server")
			})

			It("reports that some deployments are backupable and errors", func() {
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
				cmd.Dir = artifactPath
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))

				stdout := strings.Split(string(session.Out.Contents()), "\n")
				stderr := strings.Split(string(session.Err.Contents()), "\n")

				//we cant enforce the order of the output given it is random, so we assert that it contains what we expect and only those lines.
				Expect(stdout).To(ConsistOf(
					ContainSubstring(fmt.Sprintf("Pending: redis-1, redis-2, redis-3")),
					ContainSubstring("-------------------------"),
					ContainSubstring("Deployment 'redis-1' cannot be backed up."),
					"  1 error occurred:",
					"  error 1:",
					"  Deployment 'redis-1' has no backup scripts",
					ContainSubstring("Deployment 'redis-2' cannot be backed up."),
					"  1 error occurred:",
					"  error 1:",
					"  Deployment 'redis-2' has no backup scripts",
					ContainSubstring("Deployment 'redis-3' can be backed up."),
					ContainSubstring("-------------------------"),
					ContainSubstring("Successfully can be backed up: redis-3"),
					MatchRegexp("FAILED: redis-[1-2], redis-[1-2]"), //don't know which order they will fail in, so must match with regex.
					"",
				))

				Expect(stderr).To(ConsistOf(
					"2 out of 3 deployments cannot be backed up:",
					"  redis-1",
					"  redis-2",
					"",
					"Deployment 'redis-1':",
					"  1 error occurred:",
					"  error 1:",
					"  Deployment 'redis-1' has no backup scripts",
					"Deployment 'redis-2':",
					"  1 error occurred:",
					"  error 1:",
					"  Deployment 'redis-2' has no backup scripts",
					"",
					"",
				))

			})
		})
	})

	Context("when running backup", func() {
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

		It("backs up all deployments", func() {
			cmd := exec.Command(
				commandPath,
				"deployment",
				"--ca-cert", MustHaveEnv("BOSH_CA_CERT"),
				"--username", MustHaveEnv("BOSH_CLIENT"),
				"--password", MustHaveEnv("BOSH_CLIENT_SECRET"),
				"--target", MustHaveEnv("BOSH_ENVIRONMENT"),
				"--all-deployments",
				"backup",
				"--artifact-path", artifactPath,
			)
			cmd.Dir = workingDir
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			output := strings.Split(string(session.Out.Contents()), "\n")
			By("providing debug output", func() {
				Expect(output[0]).To(Equal("Starting backup..."))
				Expect(output[1]).To(ContainSubstring(fmt.Sprintf("Pending: %s, %s, %s", redis1, redis2, redis3)))
				Expect(output[2]).To(ContainSubstring("-------------------------"))
				Expect(output[3:9]).To(
					ConsistOf(
						ContainSubstring(fmt.Sprintf("Starting backup of %s, log file: %s.log", redis1, redis1)),
						ContainSubstring(fmt.Sprintf("Finished backup of %s", redis1)),
						ContainSubstring(fmt.Sprintf("Starting backup of %s, log file: %s.log", redis2, redis2)),
						ContainSubstring(fmt.Sprintf("Finished backup of %s", redis2)),
						ContainSubstring(fmt.Sprintf("Starting backup of %s, log file: %s.log", redis3, redis3)),
						ContainSubstring(fmt.Sprintf("Finished backup of %s", redis3)),
					))
				Expect(output[9]).To(ContainSubstring("-------------------------"))

				Expect(output[10]).To(ContainSubstring(fmt.Sprintf("Successfully backed up: %s, %s, %s", redis1, redis2, redis3)))
			})

			By("creating log files for each deployment", func() {
				assertBackupLogfile(redis1)
				assertBackupLogfile(redis2)
				assertBackupLogfile(redis3)
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

	Context("when running backup-cleanup", func() {
		var redisInstance1 Instance
		var redisInstance2 Instance
		var redisInstance3 Instance

		BeforeEach(func() {
			redisInstance1 = NewDeployment(redis1, "").Instance("redis", "0")
			redisInstance2 = NewDeployment(redis2, "").Instance("redis", "0")
			redisInstance3 = NewDeployment(redis3, "").Instance("redis", "0")

			session := redisInstance3.RunCommand("sudo mkdir /var/vcap/store/bbr-backup")
			Eventually(session).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			cleanupLockScriptOutput(redisInstance1)
			cleanupLockScriptOutput(redisInstance2)
			cleanupLockScriptOutput(redisInstance3)
		})

		It("cleans up all deployments", func() {
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

			cmd.Dir = artifactPath
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			By("providing debug output", func() {
				output := strings.Split(string(session.Out.Contents()), "\n")

				Expect(output[0]).To(Equal("Starting cleanup..."))
				Expect(output[1]).To(ContainSubstring(fmt.Sprintf("Pending: %s, %s, %s", redis1, redis2, redis3)))
				Expect(output[2]).To(ContainSubstring("-------------------------"))
				Expect(output[3:9]).To(
					ConsistOf(
						ContainSubstring(fmt.Sprintf("Starting cleanup of %s, log file: %s.log", redis1, redis1)),
						ContainSubstring(fmt.Sprintf("Finished cleanup of %s", redis1)),
						ContainSubstring(fmt.Sprintf("Starting cleanup of %s, log file: %s.log", redis2, redis2)),
						ContainSubstring(fmt.Sprintf("Finished cleanup of %s", redis2)),
						ContainSubstring(fmt.Sprintf("Starting cleanup of %s, log file: %s.log", redis3, redis3)),
						ContainSubstring(fmt.Sprintf("Finished cleanup of %s", redis3)),
					))
				Expect(output[9]).To(ContainSubstring("-------------------------"))

				Expect(output[10]).To(ContainSubstring(fmt.Sprintf("Successfully cleaned up: %s, %s, %s", redis1, redis2, redis3)))
			})

			By("running the post backup unlock script", func() {
				AssertPostBackupUnlock(redisInstance1)
				AssertPostBackupUnlock(redisInstance2)
				AssertPostBackupUnlock(redisInstance3)
			})

			By("cleaning up artifacts from the remote instances", func() {
				AssertArtifactsRemovedFromInstance(redisInstance3)
			})

			By("writing log files for each deployment backup-cleanup", func() {
				assertCleanupLogfile(redis1)
				assertCleanupLogfile(redis2)
				assertCleanupLogfile(redis3)
			})
		})
	})
})

func assertCleanupLogfile(deployment string) {
	files, err := filepath.Glob(filepath.Join(artifactPath, fmt.Sprintf("%s_*.log", deployment)))
	Expect(err).NotTo(HaveOccurred())
	Expect(files).To(HaveLen(1))

	logFilePath := files[0]
	Expect(filepath.Base(logFilePath)).To(MatchRegexp(fmt.Sprintf("%s_%s.log", deployment, `(\d){8}T(\d){6}Z\b`)))

	backupLogContent, err := ioutil.ReadFile(logFilePath)
	output := string(backupLogContent)

	Expect(output).To(ContainSubstring(fmt.Sprintf("INFO - Looking for scripts")))
	Expect(output).To(ContainSubstring("INFO - Running post-backup-unlock scripts..."))
	Expect(output).To(ContainSubstring("INFO - Finished running post-backup-unlock scripts."))
}

func assertBackupLogfile(deployment string) {
	files, err := filepath.Glob(filepath.Join(artifactPath, fmt.Sprintf("%s_*.log", deployment)))
	Expect(err).NotTo(HaveOccurred())
	Expect(files).To(HaveLen(1))

	logFilePath := files[0]
	Expect(filepath.Base(logFilePath)).To(MatchRegexp(fmt.Sprintf("%s_%s.log", deployment, `(\d){8}T(\d){6}Z\b`)))

	backupLogContent, err := ioutil.ReadFile(logFilePath)
	output := string(backupLogContent)

	Expect(output).To(ContainSubstring(fmt.Sprintf("Running pre-checks for backup of %s", deployment)))
	Expect(output).To(ContainSubstring("Running pre-backup-lock scripts..."))
	Expect(output).To(ContainSubstring("Running backup scripts..."))
	Expect(output).To(ContainSubstring(fmt.Sprintf("Backup created of %s", deployment)))
}

func moveBackupScript(deployment, src, dst string) {
	cmd := exec.Command(
		"bosh",
		"-d",
		deployment,
		"ssh",
		"-c",
		fmt.Sprintf("sudo mv %s %s", src, dst),
	)
	cmd.Dir = artifactPath
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
}

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
	cmd.Dir = artifactPath
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
	Expect(session.Out).To(gbytes.Say(`\b` + deployment.Name + `_(\d){8}T(\d){6}Z\b`))
}

func AssertBackupArtifactsCreated(deployment Deployment) {
	files, err := filepath.Glob(filepath.Join(artifactPath, fmt.Sprintf("%s/redis-0-redis-server.tar", BackupDirWithTimestamp(deployment.Name))))
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
