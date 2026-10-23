package command

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogFileModes(t *testing.T) {
	t.Chdir(t.TempDir())
	p, _, _, err := createLogger("now", ".", "deployment", true)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeStackTrace("failure details"); err != nil {
		t.Fatal(err)
	}
	logs, err := filepath.Glob("*.log")
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected both logs, got %v (%s)", logs, p)
	}
	for _, name := range logs {
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0077 != 0 {
			t.Fatalf("%s has mode %o", name, info.Mode().Perm())
		}
	}
}
