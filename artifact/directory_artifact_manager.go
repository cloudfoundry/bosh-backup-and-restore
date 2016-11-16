package artifact

import "os"
import "github.com/pivotal-cf/pcf-backup-and-restore/backuper"

type DirectoryArtifactManager struct{}

func (DirectoryArtifactManager) Create(name string) (backuper.Artifact, error) {
	return &DirectoryArtifact{baseDirName: name}, os.MkdirAll(name, 0700)
}

func (DirectoryArtifactManager) Open(name string) (backuper.Artifact, error) {
	_, err := os.Stat(name)
	return &DirectoryArtifact{baseDirName: name}, err
}
