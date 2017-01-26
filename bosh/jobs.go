package bosh

import (
	"fmt"
)

type Jobs []Job

func (jobs Jobs) Backupable() Jobs {
	backupableJobs := Jobs{}
	for _, job := range jobs {
		if job.HasBackup() {
			backupableJobs = append(backupableJobs, job)
		}
	}
	return backupableJobs
}
func (jobs Jobs) AnyAreBackupable() bool {
	return len(jobs.Backupable()) > 0
}

func (jobs Jobs) Restorable() Jobs {
	restorableJobs := Jobs{}
	for _, job := range jobs {
		if job.HasRestore() {
			restorableJobs = append(restorableJobs, job)
		}
	}
	return restorableJobs
}

func (jobs Jobs) PreBackupable() Jobs {
	preBackupableJobs := Jobs{}
	for _, job := range jobs {
		if job.HasPreBackup() {
			preBackupableJobs = append(preBackupableJobs, job)
		}
	}
	return preBackupableJobs
}

func (jobs Jobs) PostBackupable() Jobs {
	postBackupableJobs := Jobs{}
	for _, job := range jobs {
		if job.HasPostBackup() {
			postBackupableJobs= append(postBackupableJobs, job)
		}
	}
	return postBackupableJobs
}

func NewJobs(scripts BackupAndRestoreScripts, artifactNames map[string]string) (Jobs, error) {
	groupedByJobName := map[string]BackupAndRestoreScripts{}
	for _, script := range scripts {
		jobName, _ := script.JobName()
		existingScripts := groupedByJobName[jobName]
		groupedByJobName[jobName] = append(existingScripts, script)
	}
	var jobs []Job

	var foundNames = map[string]bool{}
	for _, name := range artifactNames {
		if foundNames[name] {
			return nil, fmt.Errorf("Multiple jobs have specified artifact name '%s'", name)
		}
		foundNames[name] = true
	}

	for jobName, jobScripts := range groupedByJobName {
		artifactName := artifactNames[jobName]

		jobs = append(jobs, NewJob(jobScripts, artifactName))
	}

	return jobs, nil
}
