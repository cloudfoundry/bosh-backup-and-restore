package instance

import (
	"fmt"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
)

func NewJob(jobScripts BackupAndRestoreScripts, metadata Metadata) Job {
	jobName := jobScripts[0].JobName()
	return Job{
		name:             jobName,
		metadata:         metadata,
		backupScript:     jobScripts.BackupOnly().firstOrBlank(),
		restoreScript:    jobScripts.RestoreOnly().firstOrBlank(),
		preBackupScript:  jobScripts.PreBackupLockOnly().firstOrBlank(),
		postBackupScript: jobScripts.PostBackupUnlockOnly().firstOrBlank(),
	}
}

type Job struct {
	name             string
	metadata         Metadata
	backupScript     Script
	preBackupScript  Script
	postBackupScript Script
	restoreScript    Script
}

func (j Job) Name() string {
	return j.name
}

func (j Job) BackupBlobName() string {
	return j.metadata.BackupName
}

func (j Job) RestoreBlobName() string {
	return j.metadata.RestoreName
}

func (j Job) BackupArtifactDirectory() string {
	return fmt.Sprintf("%s/%s", orchestrator.ArtifactDirectory, j.backupArtifactOrJobName())
}

func (j Job) RestoreArtifactDirectory() string {
	return fmt.Sprintf("%s/%s", orchestrator.ArtifactDirectory, j.restoreArtifactOrJobName())
}

func (j Job) BackupScript() Script {
	return j.backupScript
}

func (j Job) RestoreScript() Script {
	return j.restoreScript
}

func (j Job) PreBackupScript() Script {
	return j.preBackupScript
}

func (j Job) PostBackupScript() Script {
	return j.postBackupScript
}

func (j Job) HasBackup() bool {
	return j.BackupScript() != ""
}

func (j Job) HasRestore() bool {
	return j.RestoreScript() != ""
}

func (j Job) HasPreBackup() bool {
	return j.PreBackupScript() != ""
}

func (j Job) HasPostBackup() bool {
	return j.PostBackupScript() != ""
}

func (j Job) HasNamedBackupBlob() bool {
	return j.metadata.BackupName != ""
}

func (j Job) HasNamedRestoreBlob() bool {
	return j.metadata.RestoreName != ""
}

func (j Job) backupArtifactOrJobName() string {
	if j.HasNamedBackupBlob() {
		return j.BackupBlobName()
	}

	return j.name
}

func (j Job) restoreArtifactOrJobName() string {
	if j.HasNamedRestoreBlob() {
		return j.RestoreBlobName()
	}

	return j.name
}
