package readwriter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/readwriter"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/readwriter/fakes"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_readwriter.go io.ReadWriter

var _ = Describe("LogPercentageReadWriter", func() {
	Describe("LogPercentageWriter", func() {
		var fakeLogger *fakes.FakeLogger
		var fakeReadWriter *fakes.FakeReadWriter
		var logPercentageReadWriter *readwriter.LogPercentageWriter

		BeforeEach(func() {
			fakeLogger = new(fakes.FakeLogger)
			fakeReadWriter = new(fakes.FakeReadWriter)
		})

		Context("when the total size is 12 and the writer writes 4 at a time", func() {
			BeforeEach(func() {
				fakeReadWriter.WriteReturns(4, nil)
				logPercentageReadWriter = readwriter.NewLogPercentageWriter(fakeReadWriter, fakeLogger, 12, "schblam", "message")
			})

			It("logs percentage on each write", func() {
				By("logging 33% on first write")
				Expect(fakeLogger.InfoCallCount()).To(Equal(0))
				Expect(fakeReadWriter.WriteCallCount()).To(Equal(0))
				logPercentageReadWriter.Write([]byte("words"))
				Expect(fakeReadWriter.WriteCallCount()).To(Equal(1))
				Expect(fakeLogger.InfoCallCount()).To(Equal(1))
				cmd, message, args := fakeLogger.InfoArgsForCall(0)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(33))

				By("logging 66% on second write")
				logPercentageReadWriter.Write([]byte("words"))
				Expect(fakeReadWriter.WriteCallCount()).To(Equal(2))
				Expect(fakeLogger.InfoCallCount()).To(Equal(2))
				cmd, message, args = fakeLogger.InfoArgsForCall(1)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(66))
			})

			It("never logs more than 100%", func() {
				Expect(fakeLogger.InfoCallCount()).To(Equal(0))
				Expect(fakeReadWriter.WriteCallCount()).To(Equal(0))
				logPercentageReadWriter.Write([]byte("words"))
				logPercentageReadWriter.Write([]byte("words"))
				logPercentageReadWriter.Write([]byte("words"))
				logPercentageReadWriter.Write([]byte("words"))
				Expect(fakeReadWriter.WriteCallCount()).To(Equal(4))
				Expect(fakeLogger.InfoCallCount()).To(Equal(4))
				cmd, message, args := fakeLogger.InfoArgsForCall(3)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(100))
			})
		})
		Context("when writing a really big file", func() {
			BeforeEach(func() {
				fakeReadWriter.WriteReturns(1, nil)
				logPercentageReadWriter = readwriter.NewLogPercentageWriter(fakeReadWriter, fakeLogger, 100, "schblam", "message")
			})

			It("only writes logs in >5% increments", func() {
				By("not writing for the first 4%")
				Expect(fakeLogger.InfoCallCount()).To(Equal(0))
				Expect(fakeReadWriter.WriteCallCount()).To(Equal(0))
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				Expect(fakeReadWriter.WriteCallCount()).To(Equal(4))
				Expect(fakeLogger.InfoCallCount()).To(Equal(0))

				By("writing once when we hit 5%")
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				Expect(fakeReadWriter.WriteCallCount()).To(Equal(5))
				Expect(fakeLogger.InfoCallCount()).To(Equal(1))
				cmd, message, args := fakeLogger.InfoArgsForCall(0)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(5))

				By("not writing for the next 4%")
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				Expect(fakeLogger.InfoCallCount()).To(Equal(1))

				By("writing once when we hit 10%")
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				Expect(fakeLogger.InfoCallCount()).To(Equal(2))
				cmd, message, args = fakeLogger.InfoArgsForCall(1)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(10))

				By("writing once when we suddenly jump to 25%")
				fakeReadWriter.WriteReturns(15, nil)
				logPercentageReadWriter.Write([]byte("add 15 byte"))
				Expect(fakeLogger.InfoCallCount()).To(Equal(3))
				cmd, message, args = fakeLogger.InfoArgsForCall(2)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(25))

				By("not writing when we add 1% more")
				fakeReadWriter.WriteReturns(1, nil)
				logPercentageReadWriter.Write([]byte("add 1 byte"))
				Expect(fakeLogger.InfoCallCount()).To(Equal(3))
			})
		})
	})

	Describe("LogPercentageReader", func() {
		var fakeLogger *fakes.FakeLogger
		var fakeReadWriter *fakes.FakeReadWriter
		var logPercentageReadWriter *readwriter.LogPercentageReader

		BeforeEach(func() {
			fakeLogger = new(fakes.FakeLogger)
			fakeReadWriter = new(fakes.FakeReadWriter)
		})

		Context("when the total size is 12 and the reader reads 4 at a time", func() {
			BeforeEach(func() {
				fakeReadWriter.ReadReturns(4, nil)
				logPercentageReadWriter = readwriter.NewLogPercentageReader(fakeReadWriter, fakeLogger, 12, "schblam", "message")
			})

			It("logs percentage on each read", func() {
				By("logging 33% on first read")
				Expect(fakeLogger.InfoCallCount()).To(Equal(0))
				Expect(fakeReadWriter.ReadCallCount()).To(Equal(0))
				logPercentageReadWriter.Read([]byte("words"))
				Expect(fakeReadWriter.ReadCallCount()).To(Equal(1))
				Expect(fakeLogger.InfoCallCount()).To(Equal(1))
				cmd, message, args := fakeLogger.InfoArgsForCall(0)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(33))

				By("logging 66% on second read")
				logPercentageReadWriter.Read([]byte("words"))
				Expect(fakeReadWriter.ReadCallCount()).To(Equal(2))
				Expect(fakeLogger.InfoCallCount()).To(Equal(2))
				cmd, message, args = fakeLogger.InfoArgsForCall(1)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(66))
			})

			It("never logs more than 100%", func() {
				Expect(fakeLogger.InfoCallCount()).To(Equal(0))
				Expect(fakeReadWriter.ReadCallCount()).To(Equal(0))
				logPercentageReadWriter.Read([]byte("words"))
				logPercentageReadWriter.Read([]byte("words"))
				logPercentageReadWriter.Read([]byte("words"))
				logPercentageReadWriter.Read([]byte("words"))
				Expect(fakeReadWriter.ReadCallCount()).To(Equal(4))
				Expect(fakeLogger.InfoCallCount()).To(Equal(4))
				cmd, message, args := fakeLogger.InfoArgsForCall(3)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(100))
			})
		})
		Context("when writing a really big file", func() {
			BeforeEach(func() {
				fakeReadWriter.ReadReturns(1, nil)
				logPercentageReadWriter = readwriter.NewLogPercentageReader(fakeReadWriter, fakeLogger, 100, "schblam", "message")
			})

			It("only writes logs in >5% increments", func() {
				By("not writing for the first 4%")
				Expect(fakeLogger.InfoCallCount()).To(Equal(0))
				Expect(fakeReadWriter.ReadCallCount()).To(Equal(0))
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				Expect(fakeReadWriter.ReadCallCount()).To(Equal(4))
				Expect(fakeLogger.InfoCallCount()).To(Equal(0))

				By("writing once when we hit 5%")
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				Expect(fakeReadWriter.ReadCallCount()).To(Equal(5))
				Expect(fakeLogger.InfoCallCount()).To(Equal(1))
				cmd, message, args := fakeLogger.InfoArgsForCall(0)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(5))

				By("not writing for the next 4%")
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				Expect(fakeLogger.InfoCallCount()).To(Equal(1))

				By("writing once when we hit 10%")
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				Expect(fakeLogger.InfoCallCount()).To(Equal(2))
				cmd, message, args = fakeLogger.InfoArgsForCall(1)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(10))

				By("writing once when we suddenly jump to 25%")
				fakeReadWriter.ReadReturns(15, nil)
				logPercentageReadWriter.Read([]byte("add 15 byte"))
				Expect(fakeLogger.InfoCallCount()).To(Equal(3))
				cmd, message, args = fakeLogger.InfoArgsForCall(2)
				Expect(cmd).To(Equal("schblam"))
				Expect(message).To(ContainSubstring("message"))
				Expect(args[0]).To(Equal(25))

				By("not writing when we add 1% more")
				fakeReadWriter.ReadReturns(1, nil)
				logPercentageReadWriter.Read([]byte("add 1 byte"))
				Expect(fakeLogger.InfoCallCount()).To(Equal(3))
			})
		})
	})

})
