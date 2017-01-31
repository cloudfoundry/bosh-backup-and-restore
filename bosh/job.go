package bosh

import "fmt"

func NewJob(jobScripts BackupAndRestoreScripts, blobName string) Job {
	jobName, _ := jobScripts[0].JobName()
	return Job{
		name:             jobName,
		blobName:         blobName,
		backupScript:     jobScripts.BackupOnly().firstOrBlank(),
		restoreScript:    jobScripts.RestoreOnly().firstOrBlank(),
		preBackupScript:  jobScripts.PreBackupLockOnly().firstOrBlank(),
		postBackupScript: jobScripts.PostBackupUnlockOnly().firstOrBlank(),
	}
}

type Job struct {
	name             string
	blobName         string
	backupScript     Script
	preBackupScript  Script
	postBackupScript Script
	restoreScript    Script
}

func (j Job) Name() string {
	return j.name
}

func (j Job) ArtifactDirectory() string {
	return fmt.Sprintf("/var/vcap/store/backup/%s", j.artifactOrJobName())
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

func (j Job) HasNamedBlob() bool {
	return len(j.blobName) != 0
}

func (j Job) artifactOrJobName() string {
	if j.HasNamedBlob() {
		return j.blobName
	}

	return j.name
}
