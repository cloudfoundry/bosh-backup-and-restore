package orchestrator

import "io"

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
	CustomBackupBlobNames() []string
	CustomRestoreBlobNames() []string
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
		blobNames = append(blobNames, instance.CustomBackupBlobNames()...)
	}

	return blobNames
}

func (is instances) RestoreBlobNames() []string {
	var blobNames []string

	for _, instance := range is {
		blobNames = append(blobNames, instance.CustomRestoreBlobNames()...)
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
	var cleanupErrors []error
	for _, instance := range is {
		if err := instance.Cleanup(); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return ConvertErrors(cleanupErrors)
}

func (is instances) PreBackupLock() error {
	var lockErrors []error
	for _, instance := range is {
		if err := instance.PreBackupLock(); err != nil {
			lockErrors = append(lockErrors, err)
		}
	}

	return ConvertErrors(lockErrors)
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
	var unlockErrors []error
	for _, instance := range is {
		if err := instance.PostBackupUnlock(); err != nil {
			unlockErrors = append(unlockErrors, err)
		}
	}
	return ConvertErrors(unlockErrors)
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
