package all_deployments_tests

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
)

var _ = Describe("All deployments", func() {
	It("Can run pre-backup-check on all deployments", func() {
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
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))

		Expect(session.Out).To(gbytes.Say("Deployment 'redis-1' can be backed up."))
		Expect(session.Out).To(gbytes.Say("Deployment 'redis-2' can be backed up."))
		Expect(session.Out).To(gbytes.Say("Deployment 'redis-3' can be backed up."))
		Expect(session.Out).To(gbytes.Say("All 3 deployments can be backed up"))
	})
})
