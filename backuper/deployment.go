package backuper

type Deployment interface {
	IsBackupable() (bool, error)
	Backup() error
	DrainTo(Artifact) error
	Cleanup() error
}

type BoshDeployment struct {
	BoshDirector
	Logger

	instances           Instances
	backupableInstances Instances
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
	return len(backupableInstances) != 0, nil
}

func (bd *BoshDeployment) Backup() error {
	if instances, err := bd.getBackupableInstances(); err != nil {
		return err
	} else {
		return instances.Backup()
	}
}
func (bd *BoshDeployment) Cleanup() error {
	return bd.instances.Cleanup()
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
