package orderer

import "github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"

type BackupLockOrderConstraintSpecifier struct{}

func NewBackupOrderConstraintSpecifier() orderConstraintSpecifier {
	return BackupLockOrderConstraintSpecifier{}
}

func (BackupLockOrderConstraintSpecifier) Before(job orchestrator.Job) []orchestrator.JobSpecifier {
	return job.BackupShouldBeLockedBefore()
}
