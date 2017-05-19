package artifact

import (
	"archive/tar"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

const TAG = "[artifact]"

type DirectoryArtifact struct {
	orchestrator.Logger
	baseDirName string
}

func (directoryArtifact *DirectoryArtifact) logAndReturn(err error, message string, args ...interface{}) error {
	message = fmt.Sprintf(message, args...)
	directoryArtifact.Debug(TAG, "%s: %v", message, err)
	return errors.Wrap(err, message)
}
func (directoryArtifact *DirectoryArtifact) DeploymentMatches(deployment string, instances []orchestrator.Instance) (bool, error) {
	_, err := directoryArtifact.metadataExistsAndIsReadable()
	if err != nil {
		return false, directoryArtifact.logAndReturn(err, "Error checking metadata file")
	}
	meta, err := readMetadata(directoryArtifact.metadataFilename())
	if err != nil {
		return false, directoryArtifact.logAndReturn(err, "Error reading metadata file")
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

	file, err := os.Create(path.Join(directoryArtifact.baseDirName, fileName(blobIdentifier)))
	if err != nil {
		return nil, directoryArtifact.logAndReturn(err, "Error creating file %s", fileName(blobIdentifier))

	}

	return file, err
}

func (directoryArtifact *DirectoryArtifact) ReadFile(blobIdentifier orchestrator.BackupBlobIdentifier) (io.ReadCloser, error) {
	filename := directoryArtifact.instanceFilename(blobIdentifier)
	directoryArtifact.Debug(TAG, "Trying to open %s", filename)
	file, err := os.Open(filename)
	if err != nil {
		directoryArtifact.Debug(TAG, "Error reading artifact file %s", filename)
		return nil, directoryArtifact.logAndReturn(err, "Error reading artifact file %s", filename)
	}

	return file, nil
}

func (directoryArtifact *DirectoryArtifact) FetchChecksum(blobIdentifier orchestrator.BackupBlobIdentifier) (orchestrator.BackupChecksum, error) {
	metadata, err := readMetadata(directoryArtifact.metadataFilename())

	if err != nil {
		return nil, directoryArtifact.logAndReturn(err, "Error reading metadata from %s", directoryArtifact.metadataFilename())
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
		return nil, directoryArtifact.logAndReturn(err, "Error opening artifact file %v", blobIdentifier)
	}
	defer file.Close()

	tarReader := tar.NewReader(file)
	checksum := map[string]string{}
	for {
		tarHeader, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, directoryArtifact.logAndReturn(err, "Error reading tar for %s", logName(blobIdentifier))
		}
		if tarHeader.FileInfo().IsDir() || tarHeader.FileInfo().Name() == "./" {
			continue
		}

		fileShasum := sha1.New()
		if _, err := io.Copy(fileShasum, tarReader); err != nil {
			return nil, directoryArtifact.logAndReturn(err, "Error calculating sha for %s", logName(blobIdentifier))
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
			return directoryArtifact.logAndReturn(err, "Error reading metadata from %s", directoryArtifact.metadataFilename())
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
		return false, directoryArtifact.logAndReturn(err, "Error reading metadata from %s", directoryArtifact.metadataFilename())
	}

	for _, blob := range meta.MetadataForEachBlob {
		actualBlobChecksum, _ := directoryArtifact.CalculateChecksum(blob)
		if !actualBlobChecksum.Match(blob.Checksum) {
			return false, directoryArtifact.logAndReturn(err, "Can't match checksums for %s, in metadata: %v, in actual file: %v", blob.Name(), actualBlobChecksum, blob.Checksum)
		}
	}

	for _, inst := range meta.MetadataForEachInstance {
		actualInstanceChecksum, err := directoryArtifact.CalculateChecksum(inst)
		if err != nil {
			return false, directoryArtifact.logAndReturn(err, "Error calculating checksum for artifact")
		}
		if !actualInstanceChecksum.Match(inst.Checksum) {
			return false, directoryArtifact.logAndReturn(err, "Can't match checksums for %s, in metadata: %v, in actual file: %v", logName(inst), actualInstanceChecksum, inst.Checksum)
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
		return false, directoryArtifact.logAndReturn(err, "Error checking metadata exists and is readable")
	}
	return true, nil
}

func fileName(blobIdentifier orchestrator.BackupBlobIdentifier) string {
	if blobIdentifier.IsNamed() {
		return blobIdentifier.Name() + ".tar"
	}

	return blobIdentifier.Name() + "-" + blobIdentifier.Index() + ".tar"
}
