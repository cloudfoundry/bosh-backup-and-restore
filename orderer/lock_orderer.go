package orderer

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

type NoopLockOrderer struct{}

func (lo NoopLockOrderer) Order(jobs []orchestrator.Job) []orchestrator.Job {
	return jobs
}
