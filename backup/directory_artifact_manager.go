package backup

import (
	"os"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

type BackupDirectoryManager struct{}

func (BackupDirectoryManager) Create(name string, logger orchestrator.Logger) (orchestrator.Backup, error) {
	return &BackupDirectory{baseDirName: name, Logger: logger}, errors.Wrap(os.Mkdir(name, 0700), "failed creating directory")
}

func (BackupDirectoryManager) Open(name string, logger orchestrator.Logger) (orchestrator.Backup, error) {
	_, err := os.Stat(name)
	return &BackupDirectory{baseDirName: name, Logger: logger}, errors.Wrap(err, "failed opening the directory")
}

func (BackupDirectoryManager) Exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}
