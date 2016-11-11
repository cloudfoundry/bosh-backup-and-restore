package backuper

import "io"

//go:generate counterfeiter -o fakes/fake_instance.go . Instance
type Instance interface {
	Name() string
	ID() string
	IsBackupable() (bool, error)
	IsRestorable() (bool, error)
	Backup() error
	Restore() error
	Cleanup() error
	StreamBackupTo(io.Writer) error
	BackupSize() (string, error)
	BackupChecksum() (map[string]string, error)
}

type Instances []Instance

func (instances Instances) IsEmpty() bool {
	return len(instances) == 0
}
func (instances Instances) AllBackupable() (Instances, error) {
	var backupableInstances []Instance

	for _, instance := range instances {
		if backupable, err := instance.IsBackupable(); err != nil {
			return backupableInstances, err
		} else if backupable {
			backupableInstances = append(backupableInstances, instance)
		}
	}
	return backupableInstances, nil
}

func (instances Instances) AllRestoreable() (Instances, error) {
	var backupableInstances []Instance

	for _, instance := range instances {
		if backupable, err := instance.IsRestorable(); err != nil {
			return backupableInstances, err
		} else if backupable {
			backupableInstances = append(backupableInstances, instance)
		}
	}
	return backupableInstances, nil
}

func (instances Instances) Cleanup() error {
	for _, instance := range instances {
		if err := instance.Cleanup(); err != nil {
			return err
		}
	}
	return nil
}

func (instances Instances) Backup() error {
	for _, instance := range instances {
		err := instance.Backup()
		if err != nil {
			return err
		}
	}
	return nil
}

func (instances Instances) Restore() error {
	for _, instance := range instances {
		err := instance.Restore()
		if err != nil {
			return err
		}
	}
	return nil
}
