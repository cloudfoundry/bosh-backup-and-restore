package orchestrator

import (
	"io"
)

type InstanceIdentifer interface {
	Name() string
	Index() string
	ID() string
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_instance.go . Instance
type Instance interface {
	InstanceIdentifer
	IsBackupable() bool
	ArtifactDirExists() (bool, error)
	ArtifactDirCreated() bool
	MarkArtifactDirCreated()
	IsRestorable() bool
	Backup() error
	Restore() error
	Cleanup() error
	CleanupPrevious() error
	ArtifactsToBackup() []BackupArtifact
	ArtifactsToRestore() []BackupArtifact
	HasMetadataRestoreNames() bool
	Jobs() []Job
}

//counterfeiter:generate -o fakes/fake_job.go . Job
type Job interface {
	HasBackup() bool
	HasRestore() bool
	HasNamedBackupArtifact() bool
	HasNamedRestoreArtifact() bool
	BackupArtifactName() string
	RestoreArtifactName() string
	HasMetadataRestoreName() bool
	Backup() error
	PreBackupLock() error
	PostBackupUnlock(afterSuccessfulBackup bool) error
	PreRestoreLock() error
	Restore() error
	PostRestoreUnlock() error
	Name() string
	Release() string
	InstanceIdentifier() string
	BackupArtifactDirectory() string
	RestoreArtifactDirectory() string
	BackupShouldBeLockedBefore() []JobSpecifier
	RestoreShouldBeLockedBefore() []JobSpecifier
}

type JobSpecifier struct {
	Name    string
	Release string
}

type ArtifactIdentifier interface {
	InstanceName() string
	InstanceIndex() string
	InstanceID() string
	Name() string
	HasCustomName() bool
}

//counterfeiter:generate -o fakes/fake_backup_artifact.go . BackupArtifact
type BackupArtifact interface {
	ArtifactIdentifier
	Size() (string, error)
	SizeInBytes() (int, error)
	Checksum() (BackupChecksum, error)
	StreamFromRemote(io.Writer) error
	Delete() error
	StreamToRemote(io.Reader) error
}

type instances []Instance

func (is instances) IsEmpty() bool {
	return len(is) == 0
}

func (is instances) Jobs() []Job {
	var jobs []Job
	for _, instance := range is {
		jobs = append(jobs, instance.Jobs()...)
	}

	return jobs
}

func (is instances) AllBackupable() instances {
	var backupableInstances []Instance

	for _, instance := range is {
		if instance.IsBackupable() {
			backupableInstances = append(backupableInstances, instance)
		}
	}
	return backupableInstances
}

func (is instances) AllRestoreable() instances {
	var instances []Instance

	for _, instance := range is {
		if instance.IsRestorable() {
			instances = append(instances, instance)
		}
	}
	return instances
}

func (is instances) AllBackupableOrRestorable() instances {
	var instances []Instance

	for _, instance := range is {
		if instance.IsBackupable() || instance.IsRestorable() {
			instances = append(instances, instance)
		}
	}
	return instances
}

func (is instances) Cleanup() error {
	var cleanupErrors []error
	for _, instance := range is {
		if err := instance.Cleanup(); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return ConvertErrors(cleanupErrors)
}

func (is instances) CleanupPrevious() error {
	var cleanupPreviousErrors []error
	for _, instance := range is {
		if err := instance.CleanupPrevious(); err != nil {
			cleanupPreviousErrors = append(cleanupPreviousErrors, err)
		}
	}
	return ConvertErrors(cleanupPreviousErrors)
}

func (is instances) Backup() error {
	for _, instance := range is {
		err := instance.Backup()
		if err != nil {
			return err
		}
	}
	return nil
}

func (is instances) Restore() error {
	for _, instance := range is {
		err := instance.Restore()
		if err != nil {
			return err
		}
	}
	return nil
}
