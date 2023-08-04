package orchestrator

import (
	"io"
	"time"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_backup_manager.go . BackupManager
type BackupManager interface {
	Create(string, string, Logger) (Backup, error)
	Open(string, Logger) (Backup, error)
}

//counterfeiter:generate -o fakes/fake_backup.go . Backup
type Backup interface {
	GetArtifactSize(ArtifactIdentifier) (string, error)
	GetArtifactByteSize(ArtifactIdentifier) (int, error)
	CreateArtifact(ArtifactIdentifier) (io.WriteCloser, error)
	ReadArtifact(ArtifactIdentifier) (io.ReadCloser, error)
	AddChecksum(ArtifactIdentifier, BackupChecksum) error
	CreateMetadataFileWithStartTime(time.Time) error
	AddFinishTime(time.Time) error
	FetchChecksum(ArtifactIdentifier) (BackupChecksum, error)
	CalculateChecksum(ArtifactIdentifier) (BackupChecksum, error)
	DeploymentMatches(string, []Instance) (bool, error)
	SaveManifest(manifest string) error
	Valid() (bool, error)
}
