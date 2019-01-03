package director

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restores a director", func() {
	const restorePath = "/var/vcap/store/test-backup-and-restore"

	It("restores", func() {
		By("creating the artifact")
		artifactDir, err := ioutil.TempDir("", "bbr_system_test_director")
		Expect(err).NotTo(HaveOccurred())

		mustCopyBackupFixture(artifactDir)

		By("running restore")
		session := runBBRDirector("restore", "--artifact-path", artifactDir)
		Eventually(session).Should(gexec.Exit(0))

		By("ensuring data is restored")
		Eventually(runOnDirector("stat", restorePath+"/backup")).Should(gexec.Exit(0))

		By("cleaning up the restored data on the director")
		Eventually(runOnDirector("rm", "-rf", restorePath)).Should(gexec.Exit(0))

		By("cleaning up the artifact")
		Expect(os.RemoveAll(artifactDir)).To(Succeed())
	})
})
