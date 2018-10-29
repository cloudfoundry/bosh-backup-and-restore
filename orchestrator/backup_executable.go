package orchestrator

type BackupExecutable struct {
	Job
}

func NewBackupExecutable(j Job) BackupExecutable {
	return BackupExecutable{j}
}

func (e BackupExecutable) Execute() error {
	return e.Job.Backup()
}
