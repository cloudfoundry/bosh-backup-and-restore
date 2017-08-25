package orderer

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

type KahnLockOrderer struct{}

func NewKahnLockOrderer() KahnLockOrderer {
	return KahnLockOrderer{}
}

type lockingDependency struct {
	Before orchestrator.Job
	After  orchestrator.Job
}

func (lo KahnLockOrderer) Order(jobs []orchestrator.Job) []orchestrator.Job {
	var lockingDependencies = FindLockingDependencies(jobs)
	return OrderJobsUsingTheKahnAlgorithm(jobs, lockingDependencies)
}
