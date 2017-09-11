package orderer

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

type NoopLockOrderer struct{}

func NewNoopLockOrderer() NoopLockOrderer {
	return NoopLockOrderer{}
}

func (lo NoopLockOrderer) Order(jobs []orchestrator.Job) ([]orchestrator.Job, error) {
	for _, job := range jobs {
		if len(job.ShouldBeLockedBefore()) > 0 {
			return nil, fmt.Errorf("director job '%s' specifies locking dependencies, which are not allowed for director jobs", job.Name())
		}
	}
	return jobs, nil
}
