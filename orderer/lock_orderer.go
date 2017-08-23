package orderer

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

type NoopLockOrderer struct{}

func NewNoopLockOrderer() orchestrator.LockOrderer {
	return NoopLockOrderer{}
}

func (lo NoopLockOrderer) Order(jobs []orchestrator.Job) []orchestrator.Job {
	return jobs
}
