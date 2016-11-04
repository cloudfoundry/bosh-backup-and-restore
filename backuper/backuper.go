package backuper

import (
	"fmt"
	"io"
)

func New(bosh BoshDirector, artifactCreator ArtifactCreator) *Backuper {
	return &Backuper{
		BoshDirector:    bosh,
		ArtifactCreator: artifactCreator,
	}
}

//go:generate counterfeiter -o fakes/fake_artifact_creator.go . ArtifactCreator
type ArtifactCreator func(string) (Artifact, error)

//go:generate counterfeiter -o fakes/fake_artifact.go . Artifact
type Artifact interface {
	CreateFile(Instance) (io.WriteCloser, error)
	AddChecksum(Instance, string) error
}

type Backuper struct {
	BoshDirector
	ArtifactCreator
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) error {

	instances, err := b.FindInstances(deploymentName)
	if err != nil {
		return err
	}
	defer instances.Cleanup()

	backupableInstances, err := instances.AllBackupable()
	if err != nil {
		return err
	}
	if backupableInstances.IsEmpty() {
		return fmt.Errorf("Deployment '%s' has no backup scripts", deploymentName)
	}

	artifact, err := b.ArtifactCreator(deploymentName)
	if err != nil {
		return err
	}

	if err = backupableInstances.Backup(); err != nil {
		return err
	}
	//TODO: Refactor me, maybe
	for _, instance := range backupableInstances {
		writer, err := artifact.CreateFile(instance)

		if err != nil {
			return err
		}

		if err := instance.StreamBackupTo(writer); err != nil {
			return err
		}

		if err := writer.Close(); err != nil {
			return err
		}
	}
	// // PART II
	// // for _, instance := range backupableInstances {
	// // 	if artifact.ShaFor(instance) == instance.BackupSha() {
	// artifact.AddMetadata(instance, sha)
	// //
	// // 	}
	// // }
	//
	// //PART I
	// artifact.SaveMetadata()

	// _, _ = artifact.CreateFile("metadata")

	return nil
}

//go:generate counterfeiter -o fakes/fake_bosh_director.go . BoshDirector
type BoshDirector interface {
	FindInstances(deploymentName string) (Instances, error)
}
