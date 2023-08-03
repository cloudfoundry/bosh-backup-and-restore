package director

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restores a director", func() {
	const restorePath = "/var/vcap/store/test-backup-and-restore"

	It("restores", func() {
		By("running restore")
		session := runBBRDirector("restore", "--artifact-path", directorBackupFixturePath)
		Eventually(session).Should(gexec.Exit(0))

		By("ensuring data is restored")
		Eventually(runOnDirector("stat", restorePath+"/backup")).Should(gexec.Exit(0))

		By("cleaning up the restored data on the director")
		Eventually(runOnDirector("rm", "-rf", restorePath)).Should(gexec.Exit(0))
	})
})
