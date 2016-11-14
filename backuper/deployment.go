package backuper

type Deployment interface {
	IsBackupable() (bool, error)
	IsRestorable() (bool, error)
	Backup() error
	Restore() error
	DrainTo(Artifact) error
	Cleanup() error
	Instances() Instances
}

type BoshDeployment struct {
	BoshDirector
	Logger

	instances           Instances
	backupableInstances Instances
	restorableInstances Instances
}

func NewBoshDeployment(boshDirector BoshDirector, logger Logger, instances Instances) Deployment {
	return &BoshDeployment{BoshDirector: boshDirector, Logger: logger, instances: instances}
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

func (bd *BoshDeployment) DrainTo(artifact Artifact) error {
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
		if err := instance.StreamBackupTo(writer); err != nil {
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
		if err := matchChecksums(instance, localChecksum, remoteChecksum); err != nil {
			return err
		}

		artifact.AddChecksum(instance, localChecksum)
		bd.Logger.Info("", "Done.")
	}
	return nil
}

func (bd *BoshDeployment) getBackupableInstances() (Instances, error) {
	if bd.backupableInstances == nil {
		instances, err := bd.instances.AllBackupable()
		if err != nil {
			return nil, err
		}
		bd.backupableInstances = instances
	}
	return bd.backupableInstances, nil
}

func (bd *BoshDeployment) getRestoreableInstances() (Instances, error) {
	if bd.restorableInstances == nil {
		instances, err := bd.instances.AllRestoreable()
		if err != nil {
			return nil, err
		}
		bd.restorableInstances = instances
	}
	return bd.restorableInstances, nil
}
func (bd *BoshDeployment) Instances() Instances {
	return bd.instances
}
