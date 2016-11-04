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

func (d *DirectoryArtifact) CreateFile(inst Instance) (io.WriteCloser, error) {
	filename := inst.Name() + "-" + inst.ID() + ".tgz"
	return os.Create(path.Join(d.baseDirName, filename))
}
