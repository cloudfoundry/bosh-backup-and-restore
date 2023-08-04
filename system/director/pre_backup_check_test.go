package director

import (
	"github.com/onsi/gomega/gbytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("PreBackupCheck", func() {
	It("checks if the director is backupable", func() {
		session := runBBRDirector("pre-backup-check")

		Eventually(session).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("Director can be backed up"))
	})
})
