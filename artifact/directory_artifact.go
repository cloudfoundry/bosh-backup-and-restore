package artifact

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
)

const TAG = "[artifact]"

type DirectoryArtifact struct {
	orchestrator.Logger
	baseDirName string
}

func (directoryArtifact *DirectoryArtifact) DeploymentMatches(deployment string, instances []orchestrator.Instance) (bool, error) {
	_, err := directoryArtifact.metadataExistsAndIsReadable()
	if err != nil {
		directoryArtifact.Debug(TAG, "Error checking metadata file: %v", err)
		return false, err
	}
	meta, err := readMetadata(directoryArtifact.metadataFilename())
	if err != nil {
		directoryArtifact.Debug(TAG, "Error reading metadata file: %v", err)
		return false, err
	}

	for _, inst := range meta.MetadataForEachInstance {
		present := directoryArtifact.backupInstanceIsPresent(inst, instances)
		if present != true {
			directoryArtifact.Debug(TAG, "Instance %v/%v not found in %v", inst.Name(), inst.Index(), instances)
			return false, nil
		}
	}

	return true, nil
}

func (directoryArtifact *DirectoryArtifact) CreateFile(blobIdentifier orchestrator.BackupBlobIdentifier) (io.WriteCloser, error) {
	directoryArtifact.Debug(TAG, "Trying to create file %s", fileName(blobIdentifier))
	return os.Create(path.Join(directoryArtifact.baseDirName, fileName(blobIdentifier)))
}

func (directoryArtifact *DirectoryArtifact) ReadFile(blobIdentifier orchestrator.BackupBlobIdentifier) (io.ReadCloser, error) {
	filename := directoryArtifact.instanceFilename(blobIdentifier)
	directoryArtifact.Debug(TAG, "Trying to open %s", filename)
	file, err := os.Open(filename)
	if err != nil {
		directoryArtifact.Debug(TAG, "Error reading artifact file %s", filename)
		return nil, err
	}

	return file, nil
}

func (directoryArtifact *DirectoryArtifact) FetchChecksum(blobIdentifier orchestrator.BackupBlobIdentifier) (orchestrator.BackupChecksum, error) {
	metadata, err := readMetadata(directoryArtifact.metadataFilename())

	if err != nil {
		directoryArtifact.Debug(TAG, "Error reading metadata from %s %v", directoryArtifact.metadataFilename(), err)
		return nil, err
	}

	if blobIdentifier.IsNamed() {
		for _, instanceInMetadata := range metadata.MetadataForEachBlob {
			if instanceInMetadata.Name() == blobIdentifier.Name() {
				return instanceInMetadata.Checksum, nil
			}
		}
	} else {
		for _, instanceInMetadata := range metadata.MetadataForEachInstance {
			if instanceInMetadata.Index() == blobIdentifier.Index() && instanceInMetadata.Name() == blobIdentifier.Name() {
				return instanceInMetadata.Checksum, nil
			}
		}
	}

	directoryArtifact.Warn(TAG, "Checksum for %s not found in artifact", logName(blobIdentifier))
	return nil, nil
}
func logName(artifactIdentifer orchestrator.BackupBlobIdentifier) string {
	if artifactIdentifer.IsNamed() {
		return fmt.Sprintf("%s", artifactIdentifer.Name())
	}
	return fmt.Sprintf("%s/%s", artifactIdentifer.Name(), artifactIdentifer.Index())
}

func (directoryArtifact *DirectoryArtifact) CalculateChecksum(blobIdentifier orchestrator.BackupBlobIdentifier) (orchestrator.BackupChecksum, error) {
	file, err := directoryArtifact.ReadFile(blobIdentifier)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzipedReader, err := gzip.NewReader(file)
	if err != nil {
		directoryArtifact.Debug(TAG, "Cant open gzip for %s %v", logName(blobIdentifier), err)
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
			directoryArtifact.Debug(TAG, "Error reading tar for %s %v", logName(blobIdentifier), err)
			return nil, err
		}
		if tarHeader.FileInfo().IsDir() || tarHeader.FileInfo().Name() == "./" {
			continue
		}

		fileShasum := sha1.New()
		if _, err := io.Copy(fileShasum, tarReader); err != nil {
			directoryArtifact.Debug(TAG, "Error calculating sha for %s %v", logName(blobIdentifier), err)
			return nil, err
		}
		directoryArtifact.Logger.Debug(TAG, "Calculating shasum for local file %s", tarHeader.Name)
		checksum[tarHeader.Name] = fmt.Sprintf("%x", fileShasum.Sum(nil))
	}

	return checksum, nil
}
func (directoryArtifact *DirectoryArtifact) AddChecksum(blobIdentifier orchestrator.BackupBlobIdentifier, shasum orchestrator.BackupChecksum) error {
	metadata := metadata{}
	if exists, _ := directoryArtifact.metadataExistsAndIsReadable(); exists {
		var err error
		metadata, err = readMetadata(directoryArtifact.metadataFilename())
		if err != nil {
			directoryArtifact.Debug(TAG, "Error reading metadata from %s %v", directoryArtifact.metadataFilename(), err)
			return err
		}
	}

	if blobIdentifier.IsNamed() {
		metadata.MetadataForEachBlob = append(metadata.MetadataForEachBlob, blobMetadata{
			BlobName: blobIdentifier.Name(),
			Checksum: shasum,
		})
	} else {
		metadata.MetadataForEachInstance = append(metadata.MetadataForEachInstance, instanceMetadata{
			InstanceName:  blobIdentifier.Name(),
			InstanceIndex: blobIdentifier.Index(),
			Checksum:      shasum,
		})
	}

	return metadata.save(directoryArtifact.metadataFilename())
}

func (directoryArtifact *DirectoryArtifact) SaveManifest(manifest string) error {
	return ioutil.WriteFile(directoryArtifact.manifestFilename(), []byte(manifest), 0666)
}

func (directoryArtifact *DirectoryArtifact) Valid() (bool, error) {
	meta, err := readMetadata(directoryArtifact.metadataFilename())
	if err != nil {
		directoryArtifact.Debug(TAG, "Error reading metadata from %s %v", directoryArtifact.metadataFilename(), err)
		return false, err
	}

	for _, blob := range meta.MetadataForEachBlob {
		actualBlobChecksum, _ := directoryArtifact.CalculateChecksum(blob)
		if !actualBlobChecksum.Match(blob.Checksum) {
			directoryArtifact.Debug(TAG, "Can't match checksums for %s, in metadata: %v, in actual file: %v", blob.Name(), actualBlobChecksum, blob.Checksum)
			return false, nil
		}
	}

	for _, inst := range meta.MetadataForEachInstance {
		actualInstanceChecksum, err := directoryArtifact.CalculateChecksum(inst)
		if err != nil {
			return false, err
		}
		if !actualInstanceChecksum.Match(inst.Checksum) {
			directoryArtifact.Debug(TAG, "Can't match checksums for %s, in metadata: %v, in actual file: %v", logName(inst), actualInstanceChecksum, inst.Checksum)
			return false, nil
		}

	}
	return true, nil
}

func (directoryArtifact *DirectoryArtifact) backupInstanceIsPresent(backupInstance instanceMetadata, instances []orchestrator.Instance) bool {
	for _, inst := range instances {
		if inst.Index() == backupInstance.InstanceIndex && inst.Name() == backupInstance.InstanceName {
			return true
		}
	}
	return false
}

func (directoryArtifact *DirectoryArtifact) instanceFilename(blobIdentifier orchestrator.BackupBlobIdentifier) string {
	return path.Join(directoryArtifact.baseDirName, fileName(blobIdentifier))
}

func (directoryArtifact *DirectoryArtifact) metadataFilename() string {
	return path.Join(directoryArtifact.baseDirName, "metadata")
}

func (directoryArtifact *DirectoryArtifact) manifestFilename() string {
	return path.Join(directoryArtifact.baseDirName, "manifest.yml")
}
func (directoryArtifact *DirectoryArtifact) metadataExistsAndIsReadable() (bool, error) {
	_, err := os.Stat(directoryArtifact.metadataFilename())
	if err != nil {
		return false, err
	}
	return true, nil
}

func fileName(blobIdentifier orchestrator.BackupBlobIdentifier) string {
	if blobIdentifier.IsNamed() {
		return blobIdentifier.Name() + ".tgz"
	}

	return blobIdentifier.Name() + "-" + blobIdentifier.Index() + ".tgz"
}
