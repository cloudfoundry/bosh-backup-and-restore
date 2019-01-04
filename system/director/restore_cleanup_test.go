package director

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Director restore cleanup", func() {
	BeforeEach(func() {
		By("starting a restore")
		session := runBBRDirector("restore", "--artifact-path", directorBackupFixturePath)

		By("aborting the restore before it finishes")
		Eventually(session.Out).Should(gbytes.Say("Finished restoring"))
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
			By("running restore-cleanup")
			restoreCleanupSession := runBBRDirector("restore-cleanup")
			Eventually(restoreCleanupSession).Should(gexec.Exit(0))
			Eventually(restoreCleanupSession.Out).Should(gbytes.Say("'%s' cleaned up", directorHost))

			By("confirming the BBR artifact directory has been cleaned up on the director")
			sshSession := runOnDirector("ls", "-l", bbrArtifactDirectory)
			Eventually(sshSession.Err).Should(gbytes.Say("ls: cannot access '%s': No such file or directory", bbrArtifactDirectory))

			By("running restore successfully")
			restoreSession := runBBRDirector("restore", "--artifact-path", directorBackupFixturePath)
			Eventually(restoreSession).Should(gexec.Exit(0))
		})
	})

	Context("when we don't run a cleanup", func() {
		It("is in a state where subsequent restore fail", func() {
			session := runBBRDirector("restore", "--artifact-path", directorBackupFixturePath)

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("Directory /var/vcap/store/bbr-backup already exists"))
		})
	})
})
