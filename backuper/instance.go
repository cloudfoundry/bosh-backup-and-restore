package backuper

import "io"
import "github.com/hashicorp/go-multierror"

type InstanceIdentifer interface {
	Name() string
	ID() string
}

//go:generate counterfeiter -o fakes/fake_instance.go . Instance
type Instance interface {
	InstanceIdentifer
	IsBackupable() (bool, error)
	IsRestorable() (bool, error)
	PreBackupQuiesce() error
	Backup() error
	Restore() error
	Cleanup() error
	StreamBackupFromRemote(io.Writer) error
	StreamBackupToRemote(io.Reader) error
	BackupSize() (string, error)
	BackupChecksum() (BackupChecksum, error)
}

type instances []Instance

func (is instances) IsEmpty() bool {
	return len(is) == 0
}
func (is instances) AllBackupable() (instances, error) {
	var backupableInstances []Instance

	for _, instance := range is {
		if backupable, err := instance.IsBackupable(); err != nil {
			return backupableInstances, err
		} else if backupable {
			backupableInstances = append(backupableInstances, instance)
		}
	}
	return backupableInstances, nil
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

func (is instances) PreBackupQuiesce() error {
	var quiesceErrors error = nil
	for _, instance := range is {
		if err := instance.PreBackupQuiesce(); err != nil {
			quiesceErrors = multierror.Append(quiesceErrors, err)
		}
	}

	return quiesceErrors
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
