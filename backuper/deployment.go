package backuper

import "fmt"

//go:generate counterfeiter -o fakes/fake_deployment.go . Deployment
type Deployment interface {
	IsBackupable() (bool, error)
	IsRestorable() (bool, error)
	PreBackupQuiesce() error
	Backup() error
	Restore() error
	CopyRemoteBackupToLocal(Artifact) error
	CopyLocalBackupToRemote(Artifact) error
	Cleanup() error
	Instances() []Instance
}

type BoshDeployment struct {
	Logger

	instances           instances
	backupableInstances instances
	restorableInstances instances
}

func NewBoshDeployment(logger Logger, instancesArray []Instance) Deployment {
	return &BoshDeployment{Logger: logger, instances: instances(instancesArray)}
}

func (bd *BoshDeployment) IsBackupable() (bool, error) {
	bd.Logger.Info("", "Finding instances with backup scripts...")
	backupableInstances, err := bd.getBackupableInstances()
	if err != nil {
		return false, err
	}
	bd.Logger.Info("", "Done.")
	return !backupableInstances.IsEmpty(), nil
}

func (bd *BoshDeployment) PreBackupQuiesce() error {
	instances, _ := bd.getBackupableInstances()

	return instances.PreBackupQuiesce()
}

func (bd *BoshDeployment) Backup() error {
	if instances, err := bd.getBackupableInstances(); err != nil {
		return err
	} else {
		return instances.Backup()
	}
}

func (bd *BoshDeployment) Restore() error {
	if instances, err := bd.getRestoreableInstances(); err != nil {
		return err
	} else {
		return instances.Restore()
	}
}

func (bd *BoshDeployment) Cleanup() error {
	return bd.instances.Cleanup()
}

func (bd *BoshDeployment) IsRestorable() (bool, error) {
	bd.Logger.Info("", "Finding instances with restore scripts...")
	restoreableInstances, err := bd.getRestoreableInstances()
	if err != nil {
		return false, err
	}
	bd.Logger.Info("", "Done.")
	return !restoreableInstances.IsEmpty(), nil
}

func (bd *BoshDeployment) CopyRemoteBackupToLocal(artifact Artifact) error {
	instances, err := bd.getBackupableInstances()
	if err != nil {
		return err
	}
	for _, instance := range instances {
		writer, err := artifact.CreateFile(instance)

		if err != nil {
			return err
		}

		size, err := instance.BackupSize()
		if err != nil {
			return err
		}

		bd.Logger.Info("", "Copying backup -- %s uncompressed -- from %s-%s...", size, instance.Name(), instance.ID())
		if err := instance.StreamBackupFromRemote(writer); err != nil {
			return err
		}

		if err := writer.Close(); err != nil {
			return err
		}

		localChecksum, err := artifact.CalculateChecksum(instance)
		if err != nil {
			return err
		}

		remoteChecksum, err := instance.BackupChecksum()
		if err != nil {
			return err
		}
		if !localChecksum.Match(remoteChecksum) {
			return fmt.Errorf("Backup artifact is corrupted, checksum failed for %s:%s,  remote file: %s, local file: %s", instance.Name(), instance.ID(), remoteChecksum, localChecksum)
		}

		artifact.AddChecksum(instance, localChecksum)
		bd.Logger.Info("", "Done.")
	}
	return nil
}

func (bd *BoshDeployment) CopyLocalBackupToRemote(artifact Artifact) error {
	instances, err := bd.getRestoreableInstances()
	if err != nil {
		return err
	}

	for _, instance := range instances {
		reader, err := artifact.ReadFile(instance)

		if err != nil {
			return err
		}

		bd.Logger.Info("", "Copying backup to %s-%s...", instance.Name(), instance.ID())
		if err := instance.StreamBackupToRemote(reader); err != nil {
			return err
		}

		localChecksum, err := artifact.FetchChecksum(instance)
		if err != nil {
			return err
		}

		remoteChecksum, err := instance.BackupChecksum()
		if err != nil {
			return err
		}
		if !localChecksum.Match(remoteChecksum) {
			return fmt.Errorf("Backup couldn't be transfered, checksum failed for %s:%s,  remote file: %s, local file: %s", instance.Name(), instance.ID(), remoteChecksum, localChecksum)
		}
	}
	return nil
}

func (bd *BoshDeployment) getBackupableInstances() (instances, error) {
	if bd.backupableInstances == nil {
		instances, err := bd.instances.AllBackupable()
		if err != nil {
			return nil, err
		}
		bd.backupableInstances = instances
	}
	return bd.backupableInstances, nil
}

func (bd *BoshDeployment) getRestoreableInstances() (instances, error) {
	if bd.restorableInstances == nil {
		instances, err := bd.instances.AllRestoreable()
		if err != nil {
			return nil, err
		}
		bd.restorableInstances = instances
	}
	return bd.restorableInstances, nil
}
func (bd *BoshDeployment) Instances() []Instance {
	return bd.instances
}
