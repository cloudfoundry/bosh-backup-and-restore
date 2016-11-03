package backuper

import "os"

func DirectoryArtifactCreator(name string) (Artifact, error) {
	err := os.MkdirAll(name, 0700)
	if err != nil {
		panic("oh my christ")
	}
	return nil, nil
}

type DirectoryArtifact struct {
}

func (*DirectoryArtifact) CreateFile() {

}
