package orchestrator

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"

type JobPreBackupLockExecutor struct {
	Job
}

func NewJobPreBackupLockExecutable(job Job) executor.Executable {
	return JobPreBackupLockExecutor{job}
}

func (j JobPreBackupLockExecutor) Execute() error {
	return j.PreBackupLock()
}

type JobPostBackupUnlockExecutor struct {
	Job
}

func NewJobPostBackupUnlockExecutable(job Job) executor.Executable {
	return JobPostBackupUnlockExecutor{job}
}

func (j JobPostBackupUnlockExecutor) Execute() error {
	return j.PostBackupUnlock()
}

type JobPreRestoreLockExecutor struct {
	Job
}

func NewJobPreRestoreLockExecutable(job Job) executor.Executable {
	return JobPreRestoreLockExecutor{job}
}

func (j JobPreRestoreLockExecutor) Execute() error {
	return j.PreRestoreLock()
}

type JobPostRestoreUnlockExecutor struct {
	Job
}

func NewJobPostRestoreUnlockExecutable(job Job) executor.Executable {
	return JobPostRestoreUnlockExecutor{job}
}

func (j JobPostRestoreUnlockExecutor) Execute() error {
	return j.PostRestoreUnlock()
}
