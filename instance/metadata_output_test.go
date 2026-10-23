package instance

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/cloudfoundry/bosh-backup-and-restore/ssh"
)

type oversizedMetadataRunner struct {
	ssh.RemoteRunner
	written int
}

func (r *oversizedMetadataRunner) RunScriptWithEnv(_ string, _ map[string]string, _ string, out io.Writer) error {
	block := bytes.Repeat([]byte("x"), 4096)
	for range 1024 {
		n, err := out.Write(block)
		r.written += n
		if err != nil {
			return err
		}
	}
	return nil
}
func TestMetadataOutputIsBounded(t *testing.T) {
	finder := &JobFinderFromScripts{}
	runner := &oversizedMetadataRunner{}
	_, err := finder.findMetadata(InstanceIdentifier{}, Script("/var/vcap/jobs/test/bin/bbr/metadata"), runner)
	if err == nil || !strings.Contains(err.Error(), "metadata output exceeds") {
		t.Fatalf("expected output size error, got %v", err)
	}
	if runner.written > maxMetadataBytes {
		t.Fatalf("retained %d bytes", runner.written)
	}
}
