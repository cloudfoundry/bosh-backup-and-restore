package bosh_all_proxy

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"

	"fmt"
	"os/exec"
)

var _ = Describe("BoshAllProxy", func() {

	It("does not fail using BOSH_ALL_PROXY", func() {
		boshAllProxy := fmt.Sprintf(
			"ssh+socks5://%s@%s:22?private-key=%s",
			MustHaveEnv("BOSH_GW_USER"),
			MustHaveEnv("BOSH_GW_HOST"),
			MustHaveEnv("BOSH_GW_PRIVATE_KEY"),
		)

		cmd := exec.Command(
			commandPath,
			"deployment",
			"--ca-cert", MustHaveEnv("BOSH_CA_CERT"),
			"--username", MustHaveEnv("BOSH_CLIENT"),
			"--password", MustHaveEnv("BOSH_CLIENT_SECRET"),
			"--target", MustHaveEnv("BOSH_ENVIRONMENT"),
			"--deployment", ManyScriptsDeployment.Name,
			"backup",
		)
		cmd.Env = append(cmd.Env, fmt.Sprintf("BOSH_ALL_PROXY=%s", boshAllProxy))

		fmt.Println("BOSH_ALL_PROXY=", boshAllProxy, " bbr ", cmd.Args)

		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(session).Should(gexec.Exit(0))
	})
})
