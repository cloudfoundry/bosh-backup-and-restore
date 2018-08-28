package orchestrator

import (
	"fmt"
	"github.com/pkg/errors"
)

type BackupUploadExecutable struct {
	localBackup    Backup
	remoteArtifact BackupArtifact
	instance       Instance
	Logger
}

func NewBackupUploadExecutable(localBackup Backup, remoteArtifact BackupArtifact, instance Instance, logger Logger) BackupUploadExecutable {
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

	e.Logger.Info("bbr", "Copying backup for job %s on %s/%s...", e.remoteArtifact.Name(), e.instance.Name(), e.instance.Index())
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
	e.Logger.Info("bbr", "Finished copying backup for job %s on %s/%s.", e.remoteArtifact.Name(), e.instance.Name(), e.instance.Index())

	return nil
}
