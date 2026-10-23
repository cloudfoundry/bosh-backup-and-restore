package ssh

import (
	"testing"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type fixedOutputConnection struct {
	SSHConnection
	output string
}

func (c fixedOutputConnection) Run(string) ([]byte, []byte, int, error) {
	return []byte(c.output), nil, 0, nil
}

func TestRemoteOutputParsing(t *testing.T) {
	for _, out := range []string{"", " \t\n"} {
		r := SshRemoteRunner{connection: fixedOutputConnection{output: out}, logger: boshlog.NewLogger(boshlog.LevelNone)}
		if _, err := r.SizeOf("/data"); err == nil {
			t.Fatal("empty size accepted")
		}
		if _, err := r.SizeInBytes("/data"); err == nil {
			t.Fatal("empty byte size accepted")
		}
	}
	for _, out := range []string{"boom", "digest ", " filename", "digest  filename\nboom"} {
		if _, err := convertShasToMap(out); err == nil {
			t.Fatalf("malformed checksum output accepted: %q", out)
		}
	}
	got, err := convertShasToMap("digest  file with spaces\nother  second")
	if err != nil || got["file with spaces"] != "digest" || got["second"] != "other" {
		t.Fatalf("valid checksum output: %v %v", got, err)
	}
	if got, err := convertShasToMap(""); err != nil || len(got) != 0 {
		t.Fatal("empty directory should have no checksums")
	}
}
