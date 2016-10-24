package system

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Backs up a deployment", func() {
	var commandPath string
	var err error

	BeforeSuite(func() {
		SetDefaultEventuallyTimeout(60 * time.Second)
		// TODO: tests should build and upload the test release
		// By("Creating the test release")
		// RunBoshCommand(testDeploymentBoshCommand, "create-release", "--dir=../fixtures/releases/redis-test-release/", "--force")
		// By("Uploading the test release")
		// RunBoshCommand(testDeploymentBoshCommand, "upload-release", "--dir=../fixtures/releases/redis-test-release/", "--rebase")

		By("deploying the test release")
		RunBoshCommand(TestDeploymentBoshCommand(), "deploy", TestDeploymentManifest())
		By("deploying the jump box")
		RunBoshCommand(JumpBoxBoshCommand(), "deploy", JumpboxDeploymentManifest())

		os.Setenv("GOOS", "linux")
		os.Setenv("GOARCH", "amd64")
		commandPath, err = gexec.Build("github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr")
		Expect(err).NotTo(HaveOccurred())

	})
	It("backs up", func() {
		RunBoshCommand(JumpBoxSCPCommand(), commandPath, "jumpbox/0:/tmp")
		RunBoshCommand(JumpBoxSCPCommand(), MustHaveEnv("BOSH_CERT_PATH"), "jumpbox/0:/tmp/bosh.crt")

		By("running the backup command")
		session := RunCommandOnRemote(
			JumpBoxSSHCommand(),
			fmt.Sprintf(`BOSH_PASSWORD=%s /tmp/pbr --ca-cert /tmp/bosh.crt --username %s --target %s --deployment %s backup`,
				MustHaveEnv("BOSH_PASSWORD"), MustHaveEnv("BOSH_USER"), MustHaveEnv("BOSH_URL"), TestDeployment()),
		)
		Eventually(session).Should(gexec.Exit(0))
	})
	AfterSuite(func() {
		RunBoshCommand(JumpBoxBoshCommand(), "delete-deployment")
	})
})
