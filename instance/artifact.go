package instance

import (
	"fmt"
	"io"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_logger.go . Logger
type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Warn(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

func NewBackupArtifact(job orchestrator.Job, instance orchestrator.InstanceIdentifer, remoteRunner ssh.RemoteRunner, logger Logger) *Artifact {
	var name string
	if job.HasNamedBackupArtifact() {
		name = job.BackupArtifactName()
	} else {
		name = job.Name()
	}
	return &Artifact{
		isNamed:           job.HasNamedBackupArtifact(),
		artifactDirectory: job.BackupArtifactDirectory(),
		name:              name,
		instance:          instance,
		remoteRunner:      remoteRunner,
		Logger:            logger,
	}
}

func NewRestoreArtifact(job orchestrator.Job, instance orchestrator.InstanceIdentifer, remoteRunner ssh.RemoteRunner, logger Logger) *Artifact {
	var name string
	if job.HasNamedRestoreArtifact() {
		name = job.RestoreArtifactName()
	} else {
		name = job.Name()
	}
	return &Artifact{
		isNamed:           job.HasNamedRestoreArtifact(),
		artifactDirectory: job.RestoreArtifactDirectory(),
		name:              name,
		instance:          instance,
		remoteRunner:      remoteRunner,
		Logger:            logger,
	}
}

type Artifact struct {
	isNamed           bool
	index             string //nolint:unused
	artifactDirectory string
	name              string
	instance          orchestrator.InstanceIdentifer
	Logger
	remoteRunner ssh.RemoteRunner
}

func (b *Artifact) StreamFromRemote(writer io.Writer) error {
	b.Logger.Debug("bbr", "Streaming backup from instance %s/%s", b.instance.Name(), b.instance.ID()) //nolint:staticcheck
	err := b.remoteRunner.ArchiveAndDownload(b.artifactDirectory, writer)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error streaming backup from remote instance. Error: %s", err.Error()))
	}

	return nil
}

func (b *Artifact) StreamToRemote(reader io.Reader) error {
	err := b.remoteRunner.CreateDirectory(b.artifactDirectory)
	if err != nil {
		return errors.Wrap(err, "Creating backup directory on the remote failed")
	}

	b.Logger.Debug("bbr", "Streaming backup to instance %s/%s", b.instance.Name(), b.instance.ID()) //nolint:staticcheck
	return b.remoteRunner.ExtractAndUpload(reader, b.artifactDirectory)
}

func (b *Artifact) Size() (string, error) {
	b.Logger.Debug("bbr", "Calculating size of backup on %s/%s", b.instance.Name(), b.instance.ID()) //nolint:staticcheck

	size, err := b.remoteRunner.SizeOf(b.artifactDirectory)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Unable to check size of %s", b.artifactDirectory))
	}

	return size, nil
}

func (b *Artifact) SizeInBytes() (int, error) {
	size, err := b.remoteRunner.SizeInBytes(b.artifactDirectory)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("Unable to check size of %s", b.artifactDirectory))
	}
	return size, nil
}

func (b *Artifact) Checksum() (orchestrator.BackupChecksum, error) {
	b.Logger.Debug("bbr", "Calculating shasum for remote files on %s/%s", b.instance.Name(), b.instance.ID()) //nolint:staticcheck

	backupChecksum, err := b.remoteRunner.ChecksumDirectory(b.artifactDirectory)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to calculate backup checksum")
	}

	return backupChecksum, nil
}

func (b *Artifact) Delete() error {
	b.Logger.Debug("bbr", "Deleting artifact directory on %s/%s", b.instance.Name(), b.instance.ID()) //nolint:staticcheck

	err := b.remoteRunner.RemoveDirectory(b.artifactDirectory)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Unable to delete artifact directory on instance %s/%s", b.instance.Name(), b.instance.ID()))
	}

	return nil
}

func (b *Artifact) HasCustomName() bool {
	return b.isNamed
}

func (b *Artifact) Name() string {
	return b.name
}

func (b *Artifact) InstanceIndex() string {
	return b.instance.Index()
}

func (b *Artifact) InstanceID() string {
	return b.instance.ID()
}

func (b *Artifact) InstanceName() string {
	return b.instance.Name()
}
