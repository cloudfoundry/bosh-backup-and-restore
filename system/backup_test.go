package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Backs up a deployment", func() {
	var commandPath string
	var boshURL, boshUsername, boshPassword, deploymentName string

	BeforeSuite(func() {
		SetDefaultEventuallyTimeout(1 * time.Second)
		var err error
		commandPath, err = gexec.Build("github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr")
		Expect(err).NotTo(HaveOccurred())

		deploymentName = os.Getenv("BOSH_TEST_DEPLOYMENT")
		boshURL = os.Getenv("BOSH_URL")
		boshUsername = os.Getenv("BOSH_USER")
		boshPassword = os.Getenv("BOSH_PASSWORD")

		Expect(boshUsername).NotTo(BeEmpty(), "Need BOSH_USER for the test")
		Expect(boshURL).NotTo(BeEmpty(), "Need BOSH_URL for the test")
		Expect(boshPassword).NotTo(BeEmpty(), "Need BOSH_PASSWORD for the test")
	})
	var params string
	var session *gexec.Session
	JustBeforeEach(func() {
		var err error
		command := exec.Command(commandPath, strings.Split(params, " ")...)
		session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit())
	})

	Context("success", func() {
		BeforeEach(func() {
			params = fmt.Sprintf("-u %s -p %s -t %s -d %s backup", boshUsername, boshPassword, boshURL, deploymentName)
		})
		It("backs up", func() {
			Eventually(session.ExitCode()).Should(Equal(0))
		})
	})

	Context("wrong password", func() {
		BeforeEach(func() {
			params = fmt.Sprintf("-u %s -p %s -t %s -d %s backup", boshUsername, "BADPASSWORD", boshURL, deploymentName)
		})
		It("exits with error", func() {
			Eventually(session.ExitCode()).Should(Equal(1))
		})
	})

})
