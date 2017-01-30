package backuper

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
	IsPostBackupUnlockable() (bool, error)
	IsPreBackupLockable() (bool, error)
	IsRestorable() (bool, error)
	PreBackupLock() error
	Backup() error
	PostBackupUnlock() error
	Restore() error
	Cleanup() error
	StreamBackupToRemote(io.Reader) error
	BackupSize() (string, error)
	BackupChecksum() (BackupChecksum, error)
	RemoteArtifacts() []RemoteArtifact
}

//go:generate counterfeiter -o fakes/fake_remote_artifact.go . RemoteArtifact
type RemoteArtifact interface {
	Name() string
	Index() string
	ID() string //TODO: Delete me, maybe
	BackupSize() (string, error)
	BackupChecksum() (BackupChecksum, error)
	StreamBackupFromRemote(io.Writer) error
}

type instances []Instance

func (is instances) IsEmpty() bool {
	return len(is) == 0
}

func (is instances) AllBackupable() (instances, error) {
	var backupableInstances []Instance

	for _, instance := range is {
		if instance.IsBackupable() {
			backupableInstances = append(backupableInstances, instance)
		}
	}
	return backupableInstances, nil
}

func (is instances) AllPreBackupLockable() (instances, error) {
	var lockableInstances []Instance
	var findLockableErrors error = nil

	for _, instance := range is {
		if lockable, err := instance.IsPreBackupLockable(); err != nil {
			findLockableErrors = multierror.Append(err)
		} else if lockable {
			lockableInstances = append(lockableInstances, instance)
		}
	}

	return lockableInstances, findLockableErrors
}

func (is instances) AllPostBackupUnlockable() (instances, error) {
	var unlockableInstances []Instance
	var findUnlockableErrors error = nil

	for _, instance := range is {
		if unlockable, err := instance.IsPostBackupUnlockable(); err != nil {
			findUnlockableErrors = multierror.Append(err)
		} else if unlockable {
			unlockableInstances = append(unlockableInstances, instance)
		}
	}

	return unlockableInstances, findUnlockableErrors
}

func (is instances) AllRestoreable() (instances, error) {
	var backupableInstances []Instance

	for _, instance := range is {
		if backupable, err := instance.IsRestorable(); err != nil {
			return backupableInstances, err
		} else if backupable {
			backupableInstances = append(backupableInstances, instance)
		}
	}
	return backupableInstances, nil
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
