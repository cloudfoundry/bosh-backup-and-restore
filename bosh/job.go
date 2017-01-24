package bosh

import "fmt"

func NewJob(jobScripts BackupAndRestoreScripts) Job {
	jobName, _ := jobScripts[0].JobName()
	return Job{name:jobName, backupScript: firstScript(jobScripts.BackupOnly())}
}

type Job struct {
	name string
	backupScript Script
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

func (j Job) HasBackup() bool {
	return j.BackupScript() != ""
}

func firstScript(scripts BackupAndRestoreScripts) Script {
	if len(scripts) == 0 {
		return ""
	}
	return scripts[0]
}