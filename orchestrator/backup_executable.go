package orchestrator

type BackupExecutable struct {
	Instance
}

func NewBackupExecutable(i Instance) BackupExecutable {
	return BackupExecutable{i}
}

func (e BackupExecutable) Execute() error {
	return e.Instance.Backup()
}
