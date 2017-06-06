package artifact

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	sha256 "crypto/sha256"
	"time"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

const tag = "[artifact]"
const timestampFormat = "2006/01/02 15:04:05 MST"

type DirectoryArtifact struct {
	orchestrator.Logger
	baseDirName string
}

func (directoryArtifact *DirectoryArtifact) logAndReturn(err error, message string, args ...interface{}) error {
	message = fmt.Sprintf(message, args...)
	directoryArtifact.Debug(tag, "%s: %v", message, err)
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
			directoryArtifact.Debug(tag, "Instance %v/%v not found in %v", inst.Name, inst.Index, instances)
			return false, nil
		}
	}

	return true, nil
}

func (directoryArtifact *DirectoryArtifact) CreateFile(blobIdentifier orchestrator.ArtifactIdentifier) (io.WriteCloser, error) {
	directoryArtifact.Debug(tag, "Trying to create file %s", fileName(blobIdentifier))

	file, err := os.Create(path.Join(directoryArtifact.baseDirName, fileName(blobIdentifier)))
	if err != nil {
		return nil, directoryArtifact.logAndReturn(err, "Error creating file %s", fileName(blobIdentifier))

	}

	return file, err
}

func (directoryArtifact *DirectoryArtifact) ReadFile(blobIdentifier orchestrator.ArtifactIdentifier) (io.ReadCloser, error) {
	filename := directoryArtifact.instanceFilename(blobIdentifier)
	directoryArtifact.Debug(tag, "Trying to open %s", filename)
	file, err := os.Open(filename)
	if err != nil {
		directoryArtifact.Debug(tag, "Error reading artifact file %s", filename)
		return nil, directoryArtifact.logAndReturn(err, "Error reading artifact file %s", filename)
	}

	return file, nil
}

func (directoryArtifact *DirectoryArtifact) FetchChecksum(blobIdentifier orchestrator.ArtifactIdentifier) (orchestrator.BackupChecksum, error) {
	metadata, err := readMetadata(directoryArtifact.metadataFilename())

	if err != nil {
		return nil, directoryArtifact.logAndReturn(err, "Error reading metadata from %s", directoryArtifact.metadataFilename())
	}

	if blobIdentifier.HasCustomName() {
		for _, customArtifactInMetadata := range metadata.MetadataForEachBlob {
			if customArtifactInMetadata.Name == blobIdentifier.Name() {
				return customArtifactInMetadata.Checksum, nil
			}
		}
	} else {
		for _, instanceInMetadata := range metadata.MetadataForEachInstance {
			if instanceInMetadata.Index == blobIdentifier.InstanceIndex() && instanceInMetadata.Name == blobIdentifier.InstanceName() {
				for _, artifact := range instanceInMetadata.Artifacts {
					if artifact.Name == blobIdentifier.Name() {
						return artifact.Checksum, nil
					}
				}
			}
		}
	}

	directoryArtifact.Warn(tag, "Checksum for %s not found in artifact", logName(blobIdentifier))
	return nil, nil
}

func logName(artifactIdentifer orchestrator.ArtifactIdentifier) string {
	if artifactIdentifer.HasCustomName() {
		return fmt.Sprintf("%s", artifactIdentifer.Name())
	}
	return fmt.Sprintf("%s/%s", artifactIdentifer.Name(), artifactIdentifer.InstanceIndex())
}

func (directoryArtifact *DirectoryArtifact) CalculateChecksum(blobIdentifier orchestrator.ArtifactIdentifier) (orchestrator.BackupChecksum, error) {
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

		fileShasum := sha256.New()
		if _, err := io.Copy(fileShasum, tarReader); err != nil {
			return nil, directoryArtifact.logAndReturn(err, "Error calculating sha for %s", logName(blobIdentifier))
		}
		directoryArtifact.Logger.Debug(tag, "Calculating shasum for local file %s", tarHeader.Name)
		checksum[tarHeader.Name] = fmt.Sprintf("%x", fileShasum.Sum(nil))
	}

	return checksum, nil
}

func (directoryArtifact *DirectoryArtifact) AddChecksum(blobIdentifier orchestrator.ArtifactIdentifier, shasum orchestrator.BackupChecksum) error {
	if exists, err := directoryArtifact.metadataExistsAndIsReadable(); !exists {
		return directoryArtifact.logAndReturn(err, "unable to load metadata")
	}

	metadata, err := readMetadata(directoryArtifact.metadataFilename())
	if err != nil {
		return directoryArtifact.logAndReturn(err, "Error reading metadata from %s", directoryArtifact.metadataFilename())
	}

	if blobIdentifier.HasCustomName() {
		metadata.MetadataForEachBlob = append(metadata.MetadataForEachBlob, artifactMetadata{
			Name:     blobIdentifier.Name(),
			Checksum: shasum,
		})
	} else {
		instanceMetadata := metadata.findOrCreateInstanceMetadata(blobIdentifier.InstanceName(), blobIdentifier.InstanceIndex())
		instanceMetadata.Artifacts = append(instanceMetadata.Artifacts, artifactMetadata{
			Name:     blobIdentifier.Name(),
			Checksum: shasum,
		})
	}

	return metadata.save(directoryArtifact.metadataFilename())
}

func (directoryArtifact *DirectoryArtifact) CreateMetadataFileWithStartTime(startTime time.Time) error {
	exists, _ := directoryArtifact.metadataExistsAndIsReadable()
	if exists {
		message := "metadata file already exists"
		directoryArtifact.Debug(tag, "%s: %v", message, nil)
		return errors.New(message)
	}

	metadata := metadata{
		MetadataForBackupActivity: backupActivityMetadata{
			StartTime: startTime.Format(timestampFormat),
		},
	}
	metadata.save(directoryArtifact.metadataFilename())

	return nil
}

func (directoryArtifact *DirectoryArtifact) AddFinishTime(finishTime time.Time) error {
	metadata, err := readMetadata(directoryArtifact.metadataFilename())
	if err != nil {
		message := "unable to load metadata"
		directoryArtifact.Debug(tag, "%s: %v", message, nil)
		return directoryArtifact.logAndReturn(err, message)
	}

	metadata.MetadataForBackupActivity.FinishTime = finishTime.Format(timestampFormat)
	metadata.save(directoryArtifact.metadataFilename())

	return nil
}

func (directoryArtifact *DirectoryArtifact) SaveManifest(manifest string) error {
	return errors.Wrap(ioutil.WriteFile(directoryArtifact.manifestFilename(), []byte(manifest), 0666), "failed to save manifest")
}

func (directoryArtifact *DirectoryArtifact) Valid() (bool, error) {
	meta, err := readMetadata(directoryArtifact.metadataFilename())
	if err != nil {
		return false, directoryArtifact.logAndReturn(err, "Error reading metadata from %s", directoryArtifact.metadataFilename())
	}

	for _, blob := range meta.MetadataForEachBlob {
		actualBlobChecksum, _ := directoryArtifact.CalculateChecksum(makeCustomArtifactIdentifier(blob))
		if !actualBlobChecksum.Match(blob.Checksum) {
			return false, directoryArtifact.logAndReturn(err, "Can't match checksums for %s, in metadata: %v, in actual file: %v", blob.Name, actualBlobChecksum, blob.Checksum)
		}
	}

	for _, inst := range meta.MetadataForEachInstance {
		for _, artifact := range inst.Artifacts {

			actualInstanceChecksum, err := directoryArtifact.CalculateChecksum(makeDefaultArtifactIdentifier(artifact, inst))
			if err != nil {
				return false, directoryArtifact.logAndReturn(err, "Error calculating checksum for artifact")
			}

			if !actualInstanceChecksum.Match(artifact.Checksum) {
				return false, directoryArtifact.logAndReturn(err, "Can't match checksums for %s/%s %s, in metadata: %v, in actual file: %v", inst.Name, inst.Index, artifact.Name, actualInstanceChecksum, artifact.Checksum)
			}
		}
	}

	return true, nil
}

func (directoryArtifact *DirectoryArtifact) backupInstanceIsPresent(backupInstance *instanceMetadata, instances []orchestrator.Instance) bool {
	for _, inst := range instances {
		if inst.Index() == backupInstance.Index && inst.Name() == backupInstance.Name {
			return true
		}
	}
	return false
}

func (directoryArtifact *DirectoryArtifact) instanceFilename(blobIdentifier orchestrator.ArtifactIdentifier) string {
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

func fileName(blobIdentifier orchestrator.ArtifactIdentifier) string {
	if blobIdentifier.HasCustomName() {
		return customArtifactFileName(blobIdentifier.Name())
	}

	return instanceArtifactFileName(blobIdentifier.InstanceName(), blobIdentifier.InstanceIndex(), blobIdentifier.Name())
}

func instanceArtifactFileName(instanceName string, instanceIndex string, name string) string {
	return instanceName + "-" + instanceIndex + "-" + name + ".tar"
}

func customArtifactFileName(blobName string) string {
	return blobName + ".tar"
}
