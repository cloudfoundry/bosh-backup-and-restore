package backuper

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"gopkg.in/yaml.v2"
)

//go:generate counterfeiter -o fakes/fake_artifact_creator.go . ArtifactCreator
type ArtifactCreator func(string) (Artifact, error)

//go:generate counterfeiter -o fakes/fake_artifact.go . Artifact
type Artifact interface {
	CreateFile(Instance) (io.WriteCloser, error)
	ReadFile(Instance) (io.ReadCloser, error)
	AddChecksum(Instance, map[string]string) error
	CalculateChecksum(Instance) (map[string]string, error)
	DeploymentMatches(string, []Instance) (bool, error)
}

func DirectoryArtifactCreator(name string) (Artifact, error) {
	return &DirectoryArtifact{baseDirName: name}, os.MkdirAll(name, 0700)
}

func NoopArtifactCreator(name string) (Artifact, error) {
	return &DirectoryArtifact{baseDirName: name}, nil
}

type DirectoryArtifact struct {
	baseDirName string
}

type InstanceMetadata struct {
	InstanceName string            `yaml:"instance_name"`
	InstanceID   string            `yaml:"instance_id"`
	Checksum     map[string]string `yaml:"checksums"`
}

type metadata struct {
	MetadataForEachInstance []InstanceMetadata `yaml:"instances"`
}

func (d *DirectoryArtifact) DeploymentMatches(deployment string, instances []Instance) (bool, error) {
	_, err := d.metadataExistsAndIsReadable()
	if err != nil {
		return false, err
	}
	meta, err := d.readMetadata()
	if err != nil {
		return false, err
	}

	for _, inst := range meta.MetadataForEachInstance {
		present := d.backupInstanceIsPresent(inst, instances)
		if present != true {
			return false, nil
		}
	}

	return true, nil
}

func (d *DirectoryArtifact) CreateFile(inst Instance) (io.WriteCloser, error) {
	filename := inst.Name() + "-" + inst.ID() + ".tgz"
	return os.Create(path.Join(d.baseDirName, filename))
}

func (d *DirectoryArtifact) ReadFile(inst Instance) (io.ReadCloser, error) {
	filename := inst.Name() + "-" + inst.ID() + ".tgz"
	file, err := os.Open(path.Join(d.baseDirName, filename))

	if err != nil {
		return nil, err
	}

	return file, nil
}

func (d *DirectoryArtifact) CalculateChecksum(inst Instance) (map[string]string, error) {
	filename := d.instanceFilename(inst)
	file, err := os.Open(filename)
	gzipedReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	tarReader := tar.NewReader(gzipedReader)
	checksum := map[string]string{}
	for {
		tarHeader, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if tarHeader.FileInfo().IsDir() || tarHeader.FileInfo().Name() == "./" {
			continue
		}

		fileShasum := sha1.New()
		if _, err := io.Copy(fileShasum, tarReader); err != nil {
			return nil, err
		}
		checksum[tarHeader.Name] = fmt.Sprintf("%x", fileShasum.Sum(nil))
	}

	return checksum, nil
}

func (d *DirectoryArtifact) AddChecksum(inst Instance, shasum map[string]string) error {
	metadata, err := d.readMetadata()
	if err != nil {
		return err
	}

	metadata.MetadataForEachInstance = append(metadata.MetadataForEachInstance, InstanceMetadata{
		InstanceName: inst.Name(),
		InstanceID:   inst.ID(),
		Checksum:     shasum,
	})

	return d.saveMetadata(metadata)
}

func (d *DirectoryArtifact) backupInstanceIsPresent(backupInstance InstanceMetadata, instances []Instance) bool {
	for _, inst := range instances {
		if inst.ID() == backupInstance.InstanceID && inst.Name() == backupInstance.InstanceName {
			return true
		}
	}
	return false
}

func (d *DirectoryArtifact) saveMetadata(data metadata) error {
	contents, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(d.metadataFilename(), contents, 0666)
}

func (d *DirectoryArtifact) instanceFilename(inst Instance) string {
	return path.Join(d.baseDirName, inst.Name()+"-"+inst.ID()+".tgz")
}

func (d *DirectoryArtifact) metadataFilename() string {
	return path.Join(d.baseDirName, "metadata")
}

func (d *DirectoryArtifact) metadataExistsAndIsReadable() (bool, error) {
	_, err := os.Stat(d.metadataFilename())
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *DirectoryArtifact) readMetadata() (metadata, error) {
	metadata := metadata{}

	fileInfo, _ := os.Stat(d.metadataFilename())
	if fileInfo != nil {
		contents, err := ioutil.ReadFile(d.metadataFilename())
		if err != nil {
			return metadata, err
		}

		if err := yaml.Unmarshal(contents, &metadata); err != nil {
			return metadata, err
		}
	}
	return metadata, nil
}
