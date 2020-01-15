package writer_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer/fakes"
)

//go:generate counterfeiter -o fakes/fake_writer.go io.Writer

var _ = Describe("LogPercentageWriter", func() {
	var fakeLogger *fakes.FakeLogger
	var fakeWriter *fakes.FakeWriter
	var logPercentageWriter *writer.LogPercentageWriter

	BeforeEach(func() {
		fakeLogger = new(fakes.FakeLogger)
		fakeWriter = new(fakes.FakeWriter)
	})

	Context("when the total size is 12 and the writer writes 4 at a time", func() {
		BeforeEach(func() {
			fakeWriter.WriteReturns(4, nil)
			logPercentageWriter = writer.NewLogPercentageWriter(fakeWriter, fakeLogger, 12, "schblam", "message")
		})

		It("logs percentage on each write", func() {
			By("logging 33% on first write")
			Expect(fakeLogger.InfoCallCount()).To(Equal(0))
			Expect(fakeWriter.WriteCallCount()).To(Equal(0))
			logPercentageWriter.Write([]byte("words"))
			Expect(fakeWriter.WriteCallCount()).To(Equal(1))
			Expect(fakeLogger.InfoCallCount()).To(Equal(1))
			cmd, message, args := fakeLogger.InfoArgsForCall(0)
			Expect(cmd).To(Equal("schblam"))
			Expect(message).To(ContainSubstring("message"))
			Expect(args[0]).To(Equal(33))

			By("logging 66% on second write")
			logPercentageWriter.Write([]byte("words"))
			Expect(fakeWriter.WriteCallCount()).To(Equal(2))
			Expect(fakeLogger.InfoCallCount()).To(Equal(2))
			cmd, message, args = fakeLogger.InfoArgsForCall(1)
			Expect(cmd).To(Equal("schblam"))
			Expect(message).To(ContainSubstring("message"))
			Expect(args[0]).To(Equal(66))
		})

		It("never logs more than 100%", func() {
			Expect(fakeLogger.InfoCallCount()).To(Equal(0))
			Expect(fakeWriter.WriteCallCount()).To(Equal(0))
			logPercentageWriter.Write([]byte("words"))
			logPercentageWriter.Write([]byte("words"))
			logPercentageWriter.Write([]byte("words"))
			logPercentageWriter.Write([]byte("words"))
			Expect(fakeWriter.WriteCallCount()).To(Equal(4))
			Expect(fakeLogger.InfoCallCount()).To(Equal(4))
			cmd, message, args := fakeLogger.InfoArgsForCall(3)
			Expect(cmd).To(Equal("schblam"))
			Expect(message).To(ContainSubstring("message"))
			Expect(args[0]).To(Equal(100))
		})
	})
	Context("when writing a really big file", func() {
		BeforeEach(func() {
			fakeWriter.WriteReturns(1, nil)
			logPercentageWriter = writer.NewLogPercentageWriter(fakeWriter, fakeLogger, 100, "schblam", "message")
		})

		It("only writes logs in >5% increments", func() {
			Expect(fakeLogger.InfoCallCount()).To(Equal(0))
			Expect(fakeWriter.WriteCallCount()).To(Equal(0))
			logPercentageWriter.Write([]byte("words"))
			logPercentageWriter.Write([]byte("words"))
			logPercentageWriter.Write([]byte("words"))
			logPercentageWriter.Write([]byte("words"))
			Expect(fakeWriter.WriteCallCount()).To(Equal(4))
			Expect(fakeLogger.InfoCallCount()).To(Equal(0))
			logPercentageWriter.Write([]byte("words"))
			Expect(fakeWriter.WriteCallCount()).To(Equal(5))
			Expect(fakeLogger.InfoCallCount()).To(Equal(1))
			cmd, message, args := fakeLogger.InfoArgsForCall(0)
			Expect(cmd).To(Equal("schblam"))
			Expect(message).To(ContainSubstring("message"))
			Expect(args[0]).To(Equal(5))

		})
	})
})
