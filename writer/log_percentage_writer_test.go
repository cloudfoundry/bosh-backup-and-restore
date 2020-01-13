package writer_test

import (
	"bytes"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer/fakes"
)

//go:generate counterfeiter -o fakes/fake_writer.go io.Writer

var _ = Describe("Writer", func() {
	It("returns the amount written", func() {
		backingWriter := bytes.NewBuffer([]byte(""))
		logPercentageWriter := writer.NewLogPercentageWriter(backingWriter)

		n, err := logPercentageWriter.Write([]byte("four"))
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(4))

		Expect(logPercentageWriter.Count()).To(Equal(4))

		_, err = logPercentageWriter.Write([]byte("four"))
		Expect(err).NotTo(HaveOccurred())
		Expect(logPercentageWriter.Count()).To(Equal(8))
	})

	When("the write fails", func() {
		It("returns an error", func() {

			backingWriter := new(fakes.FakeWriter)
			backingWriter.WriteReturns(0, errors.New("foo"))
			logPercentageWriter := writer.NewLogPercentageWriter(backingWriter)

			_, err := logPercentageWriter.Write(nil)
			Expect(err).To(MatchError("foo"))
		})
	})
})
