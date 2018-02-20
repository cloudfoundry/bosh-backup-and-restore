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

func NewSerialJobRunner() SerialJobRunner {
	return SerialJobRunner{}
}

type SerialJobRunner struct {
}


func (jobRunner SerialJobRunner) Run(runMethod func(job Job) error, jobs [][]Job) []error {
	var errors []error
	for _, jobList := range jobs {
		for _, job := range jobList {
			if err := runMethod(job); err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
