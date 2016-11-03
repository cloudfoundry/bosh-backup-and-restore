package backuper

import (
	"io"
	"os"
	"path"
)

func DirectoryArtifactCreator(name string) (Artifact, error) {
	return &DirectoryArtifact{baseDirName: name}, os.MkdirAll(name, 0700)
}

type DirectoryArtifact struct {
	baseDirName string
}

func (d *DirectoryArtifact) CreateFile(name string) (io.WriteCloser, error) {
	return os.Create(path.Join(d.baseDirName, name))
}
