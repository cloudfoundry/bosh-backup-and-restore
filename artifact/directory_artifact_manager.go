package artifact

import (
	"os"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
)

type DirectoryArtifactManager struct{}

func (DirectoryArtifactManager) Create(name string, logger orchestrator.Logger) (orchestrator.Artifact, error) {
	return &DirectoryArtifact{baseDirName: name, Logger: logger}, os.Mkdir(name, 0700)
}

func (DirectoryArtifactManager) Open(name string, logger orchestrator.Logger) (orchestrator.Artifact, error) {
	_, err := os.Stat(name)
	return &DirectoryArtifact{baseDirName: name, Logger: logger}, err
}

func (DirectoryArtifactManager) Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
