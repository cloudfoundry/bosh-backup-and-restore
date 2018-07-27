package integration

import (
	"testing"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI Integration Suite")
}

var binary Binary

const version = "2.0"

var _ = BeforeSuite(func() {
	commandPath, err := gexec.Build(
		"github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr",
		"-ldflags",
		fmt.Sprintf("-X main.version=%s", version),
	)
	Expect(err).NotTo(HaveOccurred())
	binary = NewBinary(commandPath)
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
