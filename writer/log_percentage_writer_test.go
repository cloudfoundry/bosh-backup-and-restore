package writer_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	orchestratorFakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer/fakes"
)

//go:generate counterfeiter -o fakes/fake_writer.go io.Writer

var _ = Describe("LogPercentageWriter", func() {
	Context("when the total size is 3 and the writer writes 1 at a time", func() {
		var fakeLogger *orchestratorFakes.FakeLogger
		var fakeWriter *fakes.FakeWriter
		var logPercentageWriter *writer.LogPercentageWriter

		BeforeEach(func() {
			fakeLogger = new(orchestratorFakes.FakeLogger)
			fakeWriter = new(fakes.FakeWriter)
			fakeWriter.WriteReturns(1, nil)
			logPercentageWriter = writer.NewLogPercentageWriter(fakeWriter, fakeLogger, 3, "schblam", "message")
		})

		It("logs 33% on each write", func() {
			Expect(fakeLogger.InfoCallCount()).To(Equal(0))
			Expect(fakeWriter.WriteCallCount()).To(Equal(0))
			logPercentageWriter.Write([]byte("words"))
			Expect(fakeWriter.WriteCallCount()).To(Equal(1))
			Expect(fakeLogger.InfoCallCount()).To(Equal(1))
			cmd, message, args := fakeLogger.InfoArgsForCall(0)
			Expect(cmd).To(Equal("schblam"))
			Expect(message).To(ContainSubstring("message"))
			Expect(args[0]).To(Equal(33))
		})
	})
})
