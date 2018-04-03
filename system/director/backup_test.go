package director

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
)

var _ = Describe("Backup", func() {

	var bbrCommand string
	var artifactDir string

	directorIP := MustHaveEnv("HOST_TO_BACKUP")

	runsBBRBackupDirectorAndSucceeds := func() {
		It("backs up the director", func() {
			By("running the backup command")
			backupCommand := JumpboxInstance.RunCommandAs("vcap", bbrCommand)
			Eventually(backupCommand).Should(gexec.Exit(0))

			JumpboxInstance.AssertFilesExist([]string{
				fmt.Sprintf("%s/%s/bosh-0-test-backup-and-restore.tar", artifactDir, BackupDirWithTimestamp(directorIP)),
				fmt.Sprintf("%s/%s/bosh-0-remarkable-backup-and-restore.tar", artifactDir, BackupDirWithTimestamp(directorIP)),
				fmt.Sprintf("%s/%s/bosh-0-amazing-backup-and-restore.tar", artifactDir, BackupDirWithTimestamp(directorIP)),
			})
		})
	}

	AfterEach(func() {
		By("removing the backup")
		Eventually(JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(
				`sudo rm -rf %s/%s`,
				workspaceDir,
				directorIP,
			))).Should(gexec.Exit(0))
	})

	Context("when the operator does not specify an artifact path", func() {
		BeforeEach(func() {
			artifactDir = workspaceDir
			bbrCommand = fmt.Sprintf(
				`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s backup`,
				workspaceDir,
				directorIP,
			)
		})

		runsBBRBackupDirectorAndSucceeds()

	})

	Context("when the operator specifies a valid artifact path", func() {
		BeforeEach(func() {
			artifactDir = workspaceDir+"/artifact-dir"
			Eventually(JumpboxInstance.RunCommandAs("vcap", fmt.Sprintf("mkdir %s", artifactDir))).Should(gexec.Exit(0))

			bbrCommand = fmt.Sprintf(
				`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s backup --artifact-path %s`,
				workspaceDir,
				directorIP,
				artifactDir,
			)
		})

		runsBBRBackupDirectorAndSucceeds()
	})

	Context("when the operator specifies an artifact path that does not exist", func() {
		BeforeEach(func() {
			artifactDir = workspaceDir+"/invalid-artifact-dir"

			bbrCommand = fmt.Sprintf(
				`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s backup --artifact-path %s`,
				workspaceDir,
				directorIP,
				artifactDir,
			)
		})

		It("should fail with an artifact directory does not exist error", func() {
			session := JumpboxInstance.RunCommandAs("vcap", bbrCommand)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(string(session.Out.Contents())).Should(ContainSubstring(fmt.Sprintf("%s: no such file or directory", artifactDir)))
		})
	})

})
