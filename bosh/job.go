package bosh

import "fmt"

func NewJob(jobScripts BackupAndRestoreScripts) Job {
	jobName, _ := jobScripts[0].JobName()
	return Job{
		name:              jobName,
		backupScript:      jobScripts.BackupOnly().firstOrBlank(),
		restoreScript:     jobScripts.RestoreOnly().firstOrBlank(),
		preBackupScript:   jobScripts.PreBackupLockOnly().firstOrBlank(),
		postBackupScript:  jobScripts.PostBackupUnlockOnly().firstOrBlank(),
	}
}

type Job struct {
	name             string
	backupScript     Script
	preBackupScript  Script
	postBackupScript Script
	restoreScript    Script
}

func (j Job) Name() string {
	return j.name
}

func (j Job) ArtifactDirectory() string {
	return fmt.Sprintf("/var/vcap/store/backup/%s", j.name)
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
