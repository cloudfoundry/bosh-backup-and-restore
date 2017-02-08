package instance

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
	return !jobs.Backupable().empty()
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

func (jobs Jobs) AnyAreRestorable() bool {
	return !jobs.Restorable().empty()
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

func (jobs Jobs) AnyArePreBackupable() bool {
	return !jobs.PreBackupable().empty()
}

func (jobs Jobs) PostBackupable() Jobs {
	postBackupableJobs := Jobs{}
	for _, job := range jobs {
		if job.HasPostBackup() {
			postBackupableJobs = append(postBackupableJobs, job)
		}
	}
	return postBackupableJobs
}

func (jobs Jobs) AnyArePostBackupable() bool {
	return !jobs.PostBackupable().empty()
}

func (jobs Jobs) WithNamedBackupBlobs() Jobs {
	jobsWithNamedBlobs := Jobs{}
	for _, job := range jobs {
		if job.HasNamedBackupBlob() {
			jobsWithNamedBlobs = append(jobsWithNamedBlobs, job)
		}
	}
	return jobsWithNamedBlobs
}

func (jobs Jobs) WithNamedRestoreBlobs() Jobs {
	jobsWithNamedBlobs := Jobs{}
	for _, job := range jobs {
		if job.HasNamedRestoreBlob() {
			jobsWithNamedBlobs = append(jobsWithNamedBlobs, job)
		}
	}
	return jobsWithNamedBlobs
}

func (jobs Jobs) BackupBlobNames() []string {
	var blobNames []string

	for _, job := range jobs.WithNamedBackupBlobs() {
		blobNames = append(blobNames, job.BackupBlobName())
	}

	return blobNames
}

func (jobs Jobs) NamedBackupBlobs() []string {
	var blobNames []string

	for _, job := range jobs.WithNamedBackupBlobs() {
		blobNames = append(blobNames, job.BackupBlobName())
	}

	return blobNames
}

func (jobs Jobs) NamedRestoreBlobs() []string {
	var blobNames []string

	for _, job := range jobs.WithNamedRestoreBlobs() {
		blobNames = append(blobNames, job.RestoreBlobName())
	}

	return blobNames
}

func NewJobs(scripts BackupAndRestoreScripts, metadata map[string]Metadata) Jobs {
	groupedByJobName := map[string]BackupAndRestoreScripts{}
	for _, script := range scripts {
		jobName, _ := script.JobName()
		existingScripts := groupedByJobName[jobName]
		groupedByJobName[jobName] = append(existingScripts, script)
	}
	var jobs []Job

	for jobName, jobScripts := range groupedByJobName {
		jobs = append(jobs, NewJob(jobScripts, metadata[jobName]))
	}

	return jobs
}

func (jobs Jobs) empty() bool {
	return len(jobs) == 0
}
