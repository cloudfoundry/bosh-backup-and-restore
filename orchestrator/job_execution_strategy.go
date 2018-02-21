package orchestrator

func JobPreBackupLocker(job Job) error {
	return job.PreBackupLock()
}

func JobPreRestoreLocker(job Job) error {
	return job.PreRestoreLock()
}

func JobPostBackupUnlocker(job Job) error {
	return job.PostBackupUnlock()
}

func JobPostRestoreUnlocker(job Job) error {
	return job.PostRestoreUnlock()
}

type JobExecutionStrategy interface {
	Run(runMethod func(job Job) error, jobs [][]Job) []error
}
