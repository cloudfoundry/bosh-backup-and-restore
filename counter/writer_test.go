package counter_test

import (
	"bytes"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/counter"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/counter/fakes"
)

//go:generate counterfeiter -o fakes/fake_writer.go io.Writer

var _ = Describe("Writer", func() {
	It("returns the amount written", func() {
		writer := bytes.NewBuffer([]byte(""))
		writerCounter := counter.NewCountWriter(writer)

		n, err := writerCounter.Write([]byte("four"))
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(4))

		Expect(writerCounter.Count()).To(Equal(4))

		_, err = writerCounter.Write([]byte("four"))
		Expect(err).NotTo(HaveOccurred())
		Expect(writerCounter.Count()).To(Equal(8))
	})

	When("the write fails", func() {
		It("returns an error", func() {

			writer := new(fakes.FakeWriter)
			writer.WriteReturns(0, errors.New("foo"))
			writerCounter := counter.NewCountWriter(writer)

			_, err := writerCounter.Write(nil)
			Expect(err).To(MatchError("foo"))
		})
	})
})
