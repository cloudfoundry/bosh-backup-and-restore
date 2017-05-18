package orchestrator

import "io"
import "github.com/hashicorp/go-multierror"

type InstanceIdentifer interface {
	Name() string
	Index() string
	ID() string
}

//go:generate counterfeiter -o fakes/fake_instance.go . Instance
type Instance interface {
	InstanceIdentifer
	IsBackupable() bool
	IsPostBackupUnlockable() bool
	IsPreBackupLockable() bool
	IsRestorable() bool
	PreBackupLock() error
	Backup() error
	PostBackupUnlock() error
	Restore() error
	Cleanup() error
	BlobsToBackup() []BackupBlob
	BlobsToRestore() []BackupBlob
	CustomBlobNames() []string
	RestoreBlobNames() []string
}

type BackupBlobIdentifier interface {
	InstanceIdentifer
	IsNamed() bool
}

//go:generate counterfeiter -o fakes/fake_backup_blob.go . BackupBlob
type BackupBlob interface {
	BackupBlobIdentifier
	Size() (string, error)
	Checksum() (BackupChecksum, error)
	StreamFromRemote(io.Writer) error
	Delete() error
	StreamToRemote(io.Reader) error
}

type instances []Instance

func (is instances) IsEmpty() bool {
	return len(is) == 0
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

func (is instances) CustomBlobNames() []string {
	var blobNames []string

	for _, instance := range is {
		blobNames = append(blobNames, instance.CustomBlobNames()...)
	}

	return blobNames
}

func (is instances) RestoreBlobNames() []string {
	var blobNames []string

	for _, instance := range is {
		blobNames = append(blobNames, instance.RestoreBlobNames()...)
	}

	return blobNames
}

func (is instances) AllPreBackupLockable() instances {
	var lockableInstances []Instance

	for _, instance := range is {
		if instance.IsPreBackupLockable() {
			lockableInstances = append(lockableInstances, instance)
		}
	}

	return lockableInstances
}

func (is instances) AllPostBackupUnlockable() instances {
	var unlockableInstances []Instance

	for _, instance := range is {
		if instance.IsPostBackupUnlockable() {
			unlockableInstances = append(unlockableInstances, instance)
		}
	}

	return unlockableInstances
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

func (is instances) Cleanup() error {
	var cleanupErrors error = nil
	for _, instance := range is {
		if err := instance.Cleanup(); err != nil {
			cleanupErrors = multierror.Append(cleanupErrors, err)
		}
	}
	return cleanupErrors
}

func (is instances) PreBackupLock() error {
	var lockErrors error = nil
	for _, instance := range is {
		if err := instance.PreBackupLock(); err != nil {
			lockErrors = multierror.Append(lockErrors, err)
		}
	}

	return lockErrors
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

func (is instances) PostBackupUnlock() error {
	var unlockErrors error = nil
	for _, instance := range is {
		if err := instance.PostBackupUnlock(); err != nil {
			unlockErrors = multierror.Append(unlockErrors, err)
		}
	}
	return unlockErrors
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
