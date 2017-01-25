package bosh

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

func NewJobs(scripts BackupAndRestoreScripts) Jobs {
	groupedByJobName := map[string]BackupAndRestoreScripts{}
	for _, script := range scripts {
		jobName, _ := script.JobName()
		existingScripts := groupedByJobName[jobName]
		groupedByJobName[jobName] = append(existingScripts, script)
	}
	var jobs []Job

	for _, jobScripts := range groupedByJobName {
		jobs = append(jobs, NewJob(jobScripts))
	}
	return jobs
}
