package orchestrator

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"fmt"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o fakes/fake_artifact_copier.go . ArtifactCopier
type ArtifactCopier interface {
	DownloadBackupFromDeployment(Backup, Deployment) error
	UploadBackupToDeployment(Backup, Deployment) error
}

type artifactCopier struct {
	Logger
	executor executor.Executor
}

func NewArtifactCopier(executor executor.Executor, logger Logger) ArtifactCopier {
	return artifactCopier{
		Logger:   logger,
		executor: executor,
	}
}

func (c artifactCopier) DownloadBackupFromDeployment(localBackup Backup, deployment Deployment) error {
	instances := deployment.BackupableInstances()

	var executables []executor.Executable
	for _, instance := range instances {
		for _, remoteBackupArtifact := range instance.ArtifactsToBackup() {
			executables = append(executables, newBackupDownloadExecutable(localBackup, remoteBackupArtifact, c.Logger))
		}
	}

	errs := c.executor.Run([][]executor.Executable{executables})

	return ConvertErrors(errs)
}

type BackupDownloadExecutable struct {
	localBackup    Backup
	remoteArtifact BackupArtifact
	Logger
}

func newBackupDownloadExecutable(localBackup Backup, remoteArtifact BackupArtifact, logger Logger) BackupDownloadExecutable {
	return BackupDownloadExecutable{
		localBackup:    localBackup,
		remoteArtifact: remoteArtifact,
		Logger:         logger,
	}
}

func (e BackupDownloadExecutable) Execute() error {
	err := e.downloadBackupArtifact(e.localBackup, e.remoteArtifact)
	if err != nil {
		return err
	}

	checksum, err := e.compareChecksums(e.localBackup, e.remoteArtifact)
	if err != nil {
		return err
	}

	err = e.localBackup.AddChecksum(e.remoteArtifact, checksum)
	if err != nil {
		return err
	}

	err = e.remoteArtifact.Delete()
	if err != nil {
		return err
	}

	e.Logger.Info("bbr", "Finished validity checks -- from %s/%s...", e.remoteArtifact.InstanceName(), e.remoteArtifact.InstanceID())
	return nil
}

func (e BackupDownloadExecutable) downloadBackupArtifact(localBackup Backup, remoteBackupArtifact BackupArtifact) error {
	localBackupArtifactWriter, err := localBackup.CreateArtifact(remoteBackupArtifact)
	if err != nil {
		return err
	}

	size, err := remoteBackupArtifact.Size()
	if err != nil {
		return err
	}

	e.Logger.Info("bbr", "Copying backup -- %s uncompressed -- from %s/%s...", size, remoteBackupArtifact.InstanceName(), remoteBackupArtifact.InstanceID())
	err = remoteBackupArtifact.StreamFromRemote(localBackupArtifactWriter)
	if err != nil {
		return err
	}

	err = localBackupArtifactWriter.Close()
	if err != nil {
		return err
	}

	e.Logger.Info("bbr", "Finished copying backup -- from %s/%s...", remoteBackupArtifact.InstanceName(), remoteBackupArtifact.InstanceID())
	return nil
}

func (e BackupDownloadExecutable) compareChecksums(localBackup Backup, remoteBackupArtifact BackupArtifact) (BackupChecksum, error) {
	e.Logger.Info("bbr", "Starting validity checks -- from %s/%s...", remoteBackupArtifact.InstanceName(), remoteBackupArtifact.InstanceID())

	localChecksum, err := localBackup.CalculateChecksum(remoteBackupArtifact)
	if err != nil {
		return nil, err
	}

	remoteChecksum, err := remoteBackupArtifact.Checksum()
	if err != nil {
		return nil, err
	}

	e.Logger.Debug("bbr", "Comparing shasums")

	match, mismatchedFiles := localChecksum.Match(remoteChecksum)
	if !match {
		e.Logger.Debug("bbr", "Checksums didn't match for:")
		e.Logger.Debug("bbr", fmt.Sprintf("%v\n", mismatchedFiles))

		err = errors.Errorf(
			"Backup is corrupted, checksum failed for %s/%s %s - checksums don't match for %v. "+
				"Checksum failed for %d files in total",
			remoteBackupArtifact.InstanceName(), remoteBackupArtifact.InstanceID(), remoteBackupArtifact.Name(), getFirstTen(mismatchedFiles), len(mismatchedFiles))
		return nil, err
	}

	return localChecksum, nil
}

func (c artifactCopier) UploadBackupToDeployment(localBackup Backup, deployment Deployment) error {
	instances := deployment.RestorableInstances()

	var executables []executor.Executable
	for _, instance := range instances {
		for _, remoteBackupArtifact := range instance.ArtifactsToRestore() {
			executables = append(executables, newBackupUploadExecutable(localBackup, remoteBackupArtifact, instance, c.Logger))
		}
	}

	errs := c.executor.Run([][]executor.Executable{executables})

	return ConvertErrors(errs)
}

type BackupUploadExecutable struct {
	localBackup    Backup
	remoteArtifact BackupArtifact
	instance       Instance
	Logger
}

func newBackupUploadExecutable(localBackup Backup, remoteArtifact BackupArtifact, instance Instance, logger Logger) BackupUploadExecutable {
	return BackupUploadExecutable{
		localBackup:    localBackup,
		remoteArtifact: remoteArtifact,
		instance:       instance,
		Logger:         logger,
	}
}

func (e BackupUploadExecutable) Execute() error {
	localBackupArtifactReader, err := e.localBackup.ReadArtifact(e.remoteArtifact)
	if err != nil {
		return err
	}

	e.Logger.Info("bbr", "Copying backup to %s/%s...", e.instance.Name(), e.instance.Index())
	err = e.remoteArtifact.StreamToRemote(localBackupArtifactReader)
	if err != nil {
		return err
	}

	e.instance.MarkArtifactDirCreated()

	localChecksum, err := e.localBackup.FetchChecksum(e.remoteArtifact)
	if err != nil {
		return err
	}

	remoteChecksum, err := e.remoteArtifact.Checksum()
	if err != nil {
		return err
	}

	match, mismatchedFiles := localChecksum.Match(remoteChecksum)
	if !match {
		e.Logger.Debug("bbr", "Checksums didn't match for:")
		e.Logger.Debug("bbr", fmt.Sprintf("%v\n", mismatchedFiles))
		return errors.Errorf("Backup couldn't be transferred, checksum failed for %s/%s %s - checksums don't match for %v. Checksum failed for %d files in total",
			e.instance.Name(),
			e.instance.ID(),
			e.remoteArtifact.Name(),
			getFirstTen(mismatchedFiles),
			len(mismatchedFiles),
		)
	}
	e.Logger.Info("bbr", "Finished copying backup to %s/%s.", e.instance.Name(), e.instance.Index())

	return nil
}
