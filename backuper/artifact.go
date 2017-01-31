package backuper

import "io"

//go:generate counterfeiter -o fakes/fake_artifact_manager.go . ArtifactManager
type ArtifactManager interface {
	Create(string, Logger) (Artifact, error)
	Open(string, Logger) (Artifact, error)
	Exists(string) bool
}

//go:generate counterfeiter -o fakes/fake_artifact.go . Artifact
type Artifact interface {
	CreateFile(ArtifactIdentifer) (io.WriteCloser, error)
	ReadFile(InstanceIdentifer) (io.ReadCloser, error)
	AddChecksum(ArtifactIdentifer, BackupChecksum) error
	FetchChecksum(ArtifactIdentifer) (BackupChecksum, error)
	CalculateChecksum(ArtifactIdentifer) (BackupChecksum, error)
	DeploymentMatches(string, []Instance) (bool, error)
	SaveManifest(manifest string) error
	Valid() (bool, error)
}
