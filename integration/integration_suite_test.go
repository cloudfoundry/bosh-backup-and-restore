package integration

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var commandPath string

func runBinary(params ...string) *gexec.Session {
	command := exec.Command(commandPath, params...)
	fmt.Fprintf(GinkgoWriter, "Running command:: %v", params)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit())

	return session
}

var _ = BeforeSuite(func() {
	var err error
	commandPath, err = gexec.Build("github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr")
	Expect(err).NotTo(HaveOccurred())
})
