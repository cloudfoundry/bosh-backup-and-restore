package bosh

type Jobs []Job

func (jobs Jobs) Backupable() Jobs {
	backupableJobs :=Jobs{}
	for _, job := range jobs {
		if job.HasBackup(){
			backupableJobs  = append(backupableJobs , job)
		}
	}
	return backupableJobs
}

func NewJobs(scripts BackupAndRestoreScripts) Jobs {
	groupedByJobName := map[string]BackupAndRestoreScripts{}
	for _, script := range scripts {
		jobName,_:= script.JobName()
		existingScripts := groupedByJobName[jobName]
		groupedByJobName[jobName] = append(existingScripts, script)
	}
	var jobs []Job

	for _, jobScripts := range groupedByJobName {
		jobs = append(jobs, NewJob(jobScripts))
	}
	return jobs
}