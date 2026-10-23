package deployment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestErrorLogFileMode(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := writeStackTrace("remote error details"); err != nil {
		t.Fatal(err)
	}
	logs, err := filepath.Glob("*.log")
	if err != nil || len(logs) != 1 {
		t.Fatalf("error log: %v %v", logs, err)
	}
	info, err := os.Stat(logs[0])
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("error log has mode %o", info.Mode().Perm())
	}
}
