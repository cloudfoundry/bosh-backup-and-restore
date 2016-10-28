package backuper

type Instances []Instance

func (instances Instances) AreAnyBackupable() (bool, error) {
	for _, instance := range instances {
		if backupable, err := instance.IsBackupable(); err != nil {
			return false, err
		} else if backupable {
			return true, nil
		}
	}
	return false, nil
}

func (instances Instances) Cleanup() error {
	for _, instance := range instances {
		if err := instance.Cleanup(); err != nil {
			return err
		}
	}
	return nil
}

//go:generate counterfeiter -o fakes/fake_instance.go . Instance
type Instance interface {
	IsBackupable() (bool, error)
	Backup() error
	Cleanup() error
}
