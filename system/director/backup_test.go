package director

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
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

	boshAllProxy := fmt.Sprintf(
		"ssh+socks5://%s@%s?private-key=%s",
		MustHaveEnv("BOSH_GW_USER"),
		MustHaveEnv("BOSH_GW_HOST"),
		MustHaveEnv("BOSH_GW_PRIVATE_KEY"),
	)

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

		It("backs up the director using BOSH_ALL_PROXY", func() {
			cmd := exec.Command(
				commandPath,
				"director",
				"--username", MustHaveEnv("DIRECTOR_SSH_USERNAME"),
				"--private-key-path", MustHaveEnv("DIRECTOR_SSH_KEY_PATH"),
				"--host", MustHaveEnv("DIRECTOR_ADDRESS"),
				"pre-backup-check",
			)
			cmd.Env = append(os.Environ(), "BOSH_ALL_PROXY="+boshAllProxy)
			cmd.Stderr = GinkgoWriter
			cmd.Stdout = GinkgoWriter

			fmt.Println("BOSH_ALL_PROXY=", boshAllProxy, " bbr ", cmd.Args)

			Expect(cmd.Run()).To(Succeed())
		})
	})

	Context("when the operator specifies an artifact path that does not exist", func() {
		It("fails with an artifact directory does not exist error", func() {
			artifactDir := workspaceDir + "/invalid-artifact-dir"

			session := runBBRDirector("backup", "--artifact-path", artifactDir)

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(fmt.Sprintf("%s: no such file or directory", artifactDir)))
		})
	})
})
