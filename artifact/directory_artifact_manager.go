package artifact

import (
	"os"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

type DirectoryArtifactManager struct{}

func (DirectoryArtifactManager) Create(name string, logger orchestrator.Logger) (orchestrator.Artifact, error) {
	return &DirectoryArtifact{baseDirName: name, Logger: logger}, errors.Wrap(os.Mkdir(name, 0700), "failed creating directory")
}

func (DirectoryArtifactManager) Open(name string, logger orchestrator.Logger) (orchestrator.Artifact, error) {
	_, err := os.Stat(name)
	return &DirectoryArtifact{baseDirName: name, Logger: logger}, errors.Wrap(err, "failed opening the directory")
}

func (DirectoryArtifactManager) Exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}
