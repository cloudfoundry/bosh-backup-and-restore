package orchestrator_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("SkipStep", func() {
	var (
		logger *fakes.FakeLogger
	)

	BeforeEach(func() {
		logger = new(fakes.FakeLogger)
	})

	It("logs skipping a step", func() {
		skipStep := orchestrator.NewSkipStep(logger, "foo")

		err := skipStep.Run(nil)

		Expect(err).NotTo(HaveOccurred())
		Expect(logger.InfoCallCount()).To(Equal(1))
		_, message, params := logger.InfoArgsForCall(0)
		Expect(fmt.Sprintf(message, params...)).To(Equal("Skipping foo for deployment"))
	})
})
