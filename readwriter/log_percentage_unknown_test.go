package readwriter_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/cloudfoundry/bosh-backup-and-restore/readwriter"
	"github.com/cloudfoundry/bosh-backup-and-restore/readwriter/fakes"
)

func TestTransferWithUnknownSize(t *testing.T) {
	for _, size := range []int{0, -1} {
		logger := &fakes.FakeLogger{}
		var out bytes.Buffer
		writer := readwriter.NewLogPercentageWriter(&out, logger, size, "test", "%d")
		if n, err := writer.Write([]byte("payload")); n != 7 || err != nil {
			t.Fatalf("write: %d %v", n, err)
		}
		reader := readwriter.NewLogPercentageReader(bytes.NewBufferString("payload"), logger, size, "test", "%d")
		got, err := io.ReadAll(reader)
		if err != nil || string(got) != "payload" || out.String() != "payload" {
			t.Fatalf("transfer: %q %v", got, err)
		}
		if logger.InfoCallCount() != 0 {
			t.Fatal("reported percentage for unknown size")
		}
	}
}
