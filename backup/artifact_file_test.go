package backup

import (
	"os"
	"path/filepath"
	"testing"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func TestArtifactNamesStayInsideBackupDirectory(t *testing.T) {
	parent := t.TempDir()
	base := filepath.Join(parent, "backup")
	if err := os.Mkdir(base, 0700); err != nil {
		t.Fatal(err)
	}
	// An existing file outside the backup directory, so that only the name
	// check, not a missing file, can make the reads below fail.
	if err := os.WriteFile(filepath.Join(parent, "outside.tar"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	backup := &BackupDirectory{baseDirName: base, Logger: boshlog.NewLogger(boshlog.LevelNone)}
	identifiers := []artifactIdentifier{
		makeCustomArtifactIdentifier(artifactMetadata{Name: "../outside"}),
		makeDefaultArtifactIdentifier(artifactMetadata{Name: "job"}, &instanceMetadata{Name: "../outside", Index: "0"}),
		makeDefaultArtifactIdentifier(artifactMetadata{Name: "../../outside"}, &instanceMetadata{Name: "group", Index: "0"}),
	}
	for _, id := range identifiers {
		if f, err := backup.ReadArtifact(id); err == nil {
			f.Close() //nolint:errcheck
			t.Fatal("accepted artifact read outside backup")
		}
		if f, err := backup.CreateArtifact(id); err == nil {
			f.Close() //nolint:errcheck
			t.Fatal("accepted artifact write outside backup")
		}
		if _, err := backup.GetArtifactSize(id); err == nil {
			t.Fatal("accepted artifact size outside backup")
		}
		if _, err := backup.GetArtifactByteSize(id); err == nil {
			t.Fatal("accepted artifact byte size outside backup")
		}
	}
	data := metadata{MetadataForEachArtifact: []artifactMetadata{{Name: "../outside"}}}
	if err := data.save(backup.metadataFilename()); err != nil {
		t.Fatal(err)
	}
	if valid, err := backup.Valid(); valid || err == nil {
		t.Fatalf("invalid metadata result: %v %v", valid, err)
	}
}
