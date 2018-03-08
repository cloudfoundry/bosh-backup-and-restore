package orchestrator

type Executor interface {
	Run([][]Executable) []error
}

type Executable interface {
	Execute() error
}

func JobPreBackupLocker(job Job) error {
	return job.PreBackupLock()
}

type JobPreBackupLockExecutor struct {
	Job
}

func NewJobPreBackupLockExecutable(job Job) Executable {
	return JobPreBackupLockExecutor{job}
}

func (j JobPreBackupLockExecutor) Execute() error {
	return j.PreBackupLock()
}

type JobPostBackupUnlockExecutor struct {
	Job
}

func NewJobPostBackupUnlockExecutable(job Job) Executable {
	return JobPostBackupUnlockExecutor{job}
}

func (j JobPostBackupUnlockExecutor) Execute() error {
	return j.PostBackupUnlock()
}

type JobPreRestoreLockExecutor struct {
	Job
}

func NewJobPreRestoreLockExecutable(job Job) Executable {
	return JobPreRestoreLockExecutor{job}
}

func (j JobPreRestoreLockExecutor) Execute() error {
	return j.PreRestoreLock()
}

type JobPostRestoreUnlockExecutor struct {
	Job
}

func NewJobPostRestoreUnlockExecutable(job Job) Executable {
	return JobPostRestoreUnlockExecutor{job}
}

func (j JobPostRestoreUnlockExecutor) Execute() error {
	return j.PostRestoreUnlock()
}
