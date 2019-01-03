package director

import (
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Backup", func() {
	Context("when the operator does not specify an artifact path", func() {
		It("backs up the director", func() {
			session := runBBRDirector("backup")

			Eventually(session).Should(gexec.Exit(0))
			backupDir := mustFindBackupDir(workspaceDir)
			mustHaveFile(backupDir, "bosh-0-test-backup-and-restore.tar")
			mustHaveFile(backupDir, "bosh-0-remarkable-backup-and-restore.tar")
			mustHaveFile(backupDir, "bosh-0-amazing-backup-and-restore.tar")
		})
	})

	Context("when the operator specifies a valid artifact path", func() {
		It("backs up the director", func() {
			artifactDir, err := ioutil.TempDir("", "bbr_system_test_director")
			Expect(err).NotTo(HaveOccurred())

			session := runBBRDirector("backup", "--artifact-path", artifactDir)

			Eventually(session).Should(gexec.Exit(0))
			backupDir := mustFindBackupDir(artifactDir)
			mustHaveFile(backupDir, "bosh-0-test-backup-and-restore.tar")
			mustHaveFile(backupDir, "bosh-0-remarkable-backup-and-restore.tar")
			mustHaveFile(backupDir, "bosh-0-amazing-backup-and-restore.tar")

			Expect(os.RemoveAll(artifactDir)).To(Succeed())
		})
	})

	Context("when the operator specifies an artifact path that does not exist", func() {
		It("fails with an artifact directory does not exist error", func() {
			artifactDir := workspaceDir + "/invalid-artifact-dir"

			session := runBBRDirector("backup", "--artifact-path", artifactDir)

			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(fmt.Sprintf("%s: no such file or directory", artifactDir)))
		})
	})
})
