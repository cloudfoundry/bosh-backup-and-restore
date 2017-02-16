package orchestrator

import "fmt"

//go:generate counterfeiter -o fakes/fake_deployment.go . Deployment
type Deployment interface {
	IsBackupable() bool
	HasValidBackupMetadata() bool
	IsRestorable() bool
	PreBackupLock() error
	Backup() error
	PostBackupUnlock() error
	Restore() error
	CopyRemoteBackupToLocal(Artifact) error
	CopyLocalBackupToRemote(Artifact) error
	Cleanup() error
	Instances() []Instance
}

type BoshDeployment struct {
	Logger

	instances instances
}

func NewBoshDeployment(logger Logger, instancesArray []Instance) Deployment {
	return &BoshDeployment{Logger: logger, instances: instances(instancesArray)}
}

func (bd *BoshDeployment) IsBackupable() bool {
	bd.Logger.Info("", "Finding instances with backup scripts...")
	backupableInstances := bd.instances.AllBackupable()
	bd.Logger.Info("", "Done.")
	return !backupableInstances.IsEmpty()
}

func (bd *BoshDeployment) HasValidBackupMetadata() bool {
	names := bd.instances.CustomBlobNames()

	uniqueNames := map[string]bool{}
	for _, name := range names {
		if _, found := uniqueNames[name]; found {
			return false
		}
		uniqueNames[name] = true
	}
	return true
}

func (bd *BoshDeployment) PreBackupLock() error {
	bd.Logger.Info("", "Running pre-backup scripts...")
	err := bd.instances.AllPreBackupLockable().PreBackupLock()
	bd.Logger.Info("", "Done.")
	return err
}

func (bd *BoshDeployment) Backup() error {
	bd.Logger.Info("", "Running backup scripts...")
	return bd.instances.AllBackupable().Backup()
}

func (bd *BoshDeployment) PostBackupUnlock() error {
	bd.Logger.Info("", "Running post-backup scripts...")
	err := bd.instances.AllPostBackupUnlockable().PostBackupUnlock()
	bd.Logger.Info("", "Done.")
	return err
}

func (bd *BoshDeployment) Restore() error {
	bd.Logger.Info("", "Running restore scripts...")
	return bd.instances.AllRestoreable().Restore()
}

func (bd *BoshDeployment) Cleanup() error {
	return bd.instances.Cleanup()
}

func (bd *BoshDeployment) IsRestorable() bool {
	bd.Logger.Info("", "Finding instances with restore scripts...")
	restoreableInstances := bd.instances.AllRestoreable()
	return !restoreableInstances.IsEmpty()
}

func (bd *BoshDeployment) CopyRemoteBackupToLocal(artifact Artifact) error {
	instances := bd.instances.AllBackupable()
	for _, instance := range instances {
		for _, remoteArtifact := range instance.BlobsToBackup() {
			writer, err := artifact.CreateFile(remoteArtifact)

			if err != nil {
				return err
			}

			size, err := remoteArtifact.Size()
			if err != nil {
				return err
			}

			bd.Logger.Info("", "Copying backup -- %s uncompressed -- from %s/%s...", size, instance.Name(), instance.ID())
			if err := remoteArtifact.StreamFromRemote(writer); err != nil {
				return err
			}

			if err := writer.Close(); err != nil {
				return err
			}

			localChecksum, err := artifact.CalculateChecksum(remoteArtifact)
			if err != nil {
				return err
			}

			remoteChecksum, err := remoteArtifact.Checksum()
			if err != nil {
				return err
			}
			if !localChecksum.Match(remoteChecksum) {
				return fmt.Errorf("Backup artifact is corrupted, checksum failed for %s/%s %s,  remote file: %s, local file: %s", instance.Name(), instance.ID(), remoteArtifact.Name(), remoteChecksum, localChecksum)
			}

			artifact.AddChecksum(remoteArtifact, localChecksum)

			err = remoteArtifact.Delete()
			if err != nil {
				return err
			}

			bd.Logger.Info("", "Done.")
		}
	}
	return nil
}

func (bd *BoshDeployment) CopyLocalBackupToRemote(artifact Artifact) error {
	instances := bd.instances.AllRestoreable()

	for _, instance := range instances {
		for _, blob := range instance.BlobsToRestore() {
			reader, err := artifact.ReadFile(blob)

			if err != nil {
				return err
			}

			bd.Logger.Info("", "Copying backup to %s/%s...", blob.Name(), blob.ID())
			if err := blob.StreamToRemote(reader); err != nil {
				return err
			}

			localChecksum, err := artifact.FetchChecksum(blob)
			if err != nil {
				return err
			}

			remoteChecksum, err := blob.Checksum()
			if err != nil {
				return err
			}
			if !localChecksum.Match(remoteChecksum) {
				return fmt.Errorf("Backup couldn't be transfered, checksum failed for %s/%s %s,  remote file: %s, local file: %s", instance.Name(), instance.ID(), blob.Name(), remoteChecksum, localChecksum)
			}
			bd.Logger.Info("", "Done.")
		}
	}
	return nil
}

func (bd *BoshDeployment) Instances() []Instance {
	return bd.instances
}
