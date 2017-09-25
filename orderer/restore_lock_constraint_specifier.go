package orderer

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

type RestoreLockOrderConstraintSpecifier struct{}

func NewRestoreOrderConstraintSpecifier() orderConstraintSpecifier {
	return RestoreLockOrderConstraintSpecifier{}
}

func (RestoreLockOrderConstraintSpecifier) Before(job orchestrator.Job) []orchestrator.JobSpecifier {
	return job.RestoreShouldBeLockedBefore()
}
