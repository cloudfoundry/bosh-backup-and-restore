package orderer

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

type NoopLockOrderer struct{}

func NewNoopLockOrderer() NoopLockOrderer {
	return NoopLockOrderer{}
}

func (lo NoopLockOrderer) Order(jobs []orchestrator.Job) ([]orchestrator.Job, error) {
	return jobs, nil
}
