package director

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/bosh-backup-and-restore/testcluster"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restore Cleanup", func() {
	var cleanupWorkspace string

	Context("when director has a backup artifact", func() {
		var session *gexec.Session
		var directorInstance *testcluster.Instance
		var directorAddress string

		BeforeEach(func() {
			cleanupWorkspace, _ = os.MkdirTemp(".", "cleanup-workspace-") //nolint:errcheck

			directorInstance = testcluster.NewInstance()
			directorInstance.CreateUser("foobar", readFile(pathToPublicKeyFile))
			directorAddress = directorInstance.Address()

			directorInstance.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", ``)
			directorInstance.CreateDir("/var/vcap/store/bbr-backup")
		})

		JustBeforeEach(func() {
			session = binary.Run(
				cleanupWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))},
				"director",
				"--host", directorAddress,
				"--username", "foobar",
				"--private-key-path", pathToPrivateKeyFile,
				"--debug",
				"restore-cleanup",
			)
		})

		AfterEach(func() {
			directorInstance.DieInBackground()
			Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
		})

		It("successfully cleans up the director after a failed restore", func() {
			Eventually(session.ExitCode()).Should(Equal(0))
			Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
		})
	})
})
