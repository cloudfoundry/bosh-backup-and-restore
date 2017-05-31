package orchestrator

import (
	"io"
	"time"
)

//go:generate counterfeiter -o fakes/fake_artifact_manager.go . ArtifactManager
type ArtifactManager interface {
	Create(string, Logger) (Artifact, error)
	Open(string, Logger) (Artifact, error)
	Exists(string) bool
}

//go:generate counterfeiter -o fakes/fake_artifact.go . Artifact
type Artifact interface {
	CreateFile(BackupBlobIdentifier) (io.WriteCloser, error)
	ReadFile(BackupBlobIdentifier) (io.ReadCloser, error)
	AddChecksum(BackupBlobIdentifier, BackupChecksum) error
	CreateMetadataFileWithStartTime(time.Time) error
	AddFinishTime(time.Time) error
	FetchChecksum(BackupBlobIdentifier) (BackupChecksum, error)
	CalculateChecksum(BackupBlobIdentifier) (BackupChecksum, error)
	DeploymentMatches(string, []Instance) (bool, error)
	SaveManifest(manifest string) error
	Valid() (bool, error)
}
