package backuper

import "io"

//go:generate counterfeiter -o fakes/fake_artifact_creator.go . ArtifactCreator
type ArtifactCreator func(string) (Artifact, error)

//go:generate counterfeiter -o fakes/fake_artifact_manager.go . ArtifactManager
type ArtifactManager interface {
	Create(string) (Artifact, error)
	Open(string) (Artifact, error)
}

//go:generate counterfeiter -o fakes/fake_artifact.go . Artifact
type Artifact interface {
	CreateFile(Instance) (io.WriteCloser, error)
	ReadFile(Instance) (io.ReadCloser, error)
	AddChecksum(Instance, map[string]string) error
	CalculateChecksum(Instance) (map[string]string, error)
	DeploymentMatches(string, []Instance) (bool, error)
	SaveManifest(manifest string) error
}
