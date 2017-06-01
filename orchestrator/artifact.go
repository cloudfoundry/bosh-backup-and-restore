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
	CreateFile(ArtifactIdentifier) (io.WriteCloser, error)
	ReadFile(ArtifactIdentifier) (io.ReadCloser, error)
	AddChecksum(ArtifactIdentifier, BackupChecksum) error
	CreateMetadataFileWithStartTime(time.Time) error
	AddFinishTime(time.Time) error
	FetchChecksum(ArtifactIdentifier) (BackupChecksum, error)
	CalculateChecksum(ArtifactIdentifier) (BackupChecksum, error)
	DeploymentMatches(string, []Instance) (bool, error)
	SaveManifest(manifest string) error
	Valid() (bool, error)
}
