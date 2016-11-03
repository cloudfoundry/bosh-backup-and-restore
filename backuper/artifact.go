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

func (d *DirectoryArtifact) CreateFile(name string, contents io.Reader) error {
	var file *os.File
	var err error

	if file, err = os.Create(path.Join(d.baseDirName, name)); err != nil {
		return err
	}

	if _, err = io.Copy(file, contents); err != nil {
		return err
	}

	return file.Close()
}
