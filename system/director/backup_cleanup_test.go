package director

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Director backup cleanup", func() {
	BeforeEach(func() {
		By("starting a backup")
		session := runBBRDirector("backup")

		By("aborting the backup before it finishes")
		Eventually(session.Out).Should(gbytes.Say("Finished backing up"))
		session.Kill().Wait(1 * time.Second)
		Expect(session).To(gexec.Exit())
		_, err := GinkgoWriter.Write([]byte("----------\n"))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		By("cleaning up the BBR artifact directory on the director")
		Eventually(runOnDirector("rm", "-rf", bbrArtifactDirectory)).Should(gexec.Exit(0))
	})

	Context("When we run cleanup", func() {
		It("succeeds", func() {
			By("running backup-cleanup")
			backupCleanupSession := runBBRDirector("backup-cleanup")
			Eventually(backupCleanupSession).Should(gexec.Exit(0))
			Eventually(backupCleanupSession.Out).Should(gbytes.Say("'%s' cleaned up", directorHost))

			By("confirming the BBR artifact directory has been cleaned up on the director")
			sshSession := runOnDirector("ls", "-l", bbrArtifactDirectory)
			Eventually(sshSession.Err).Should(gbytes.Say("ls: cannot access '%s': No such file or directory", bbrArtifactDirectory))

			By("running backup successfully")
			backupSession := runBBRDirector("backup")
			Eventually(backupSession).Should(gexec.Exit(0))
		})
	})

	Context("when we don't run a cleanup", func() {
		It("is in a state where subsequent backups fail", func() {
			session := runBBRDirector("backup")

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("Directory %s already exists", bbrArtifactDirectory))
		})
	})
})
