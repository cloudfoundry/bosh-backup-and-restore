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
	boshAllProxy := fmt.Sprintf(
		"ssh+socks5://%s@%s?private-key=%s",
		MustHaveEnv("BOSH_GW_USER"),
		MustHaveEnv("BOSH_GW_HOST"),
		MustHaveEnv("BOSH_GW_PRIVATE_KEY"),
	)

	It("backs up the deployment using BOSH_ALL_PROXY", func() {
		cmd := exec.Command(
			commandPath,
			"deployment",
			"--ca-cert", MustHaveEnv("BOSH_CA_CERT"),
			"--username", MustHaveEnv("BOSH_CLIENT"),
			"--password", MustHaveEnv("BOSH_CLIENT_SECRET"),
			"--target", MustHaveEnv("BOSH_ENVIRONMENT"),
			"--deployment", "many-bbr-scripts",
			"backup",
		)
		cmd.Env = append(cmd.Env, fmt.Sprintf("BOSH_ALL_PROXY=%s", boshAllProxy))

		fmt.Println("BOSH_ALL_PROXY=", boshAllProxy, " bbr ", cmd.Args)

		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(session).Should(gexec.Exit(0))
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
		cmd.Env = append(cmd.Env, fmt.Sprintf("BOSH_ALL_PROXY=%s", boshAllProxy))

		fmt.Println("BOSH_ALL_PROXY=", boshAllProxy, " bbr ", cmd.Args)

		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(session).Should(gexec.Exit(0))
	})
})
