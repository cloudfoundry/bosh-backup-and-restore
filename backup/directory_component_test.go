package backup_test

import (
	"testing"

	"github.com/cloudfoundry/bosh-backup-and-restore/backup"
)

func TestBackupDirectoryRejectsInvalidNames(t *testing.T) {
	manager := backup.BackupDirectoryManager{}
	for _, name := range []string{"../outside", "/absolute", "..", ".", `..\outside`, ""} {
		if _, err := manager.Create(t.TempDir(), name, nil); err == nil {
			t.Fatalf("accepted backup directory %q", name)
		}
	}
	if _, err := manager.Create(t.TempDir(), "normal-deployment_20261023T000000Z", nil); err != nil {
		t.Fatal(err)
	}
}
