package orderer

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

type DirectorLockOrderer struct{}

func NewDirectorLockOrderer() DirectorLockOrderer {
	return DirectorLockOrderer{}
}

func (lo DirectorLockOrderer) Order(jobs []orchestrator.Job) ([]orchestrator.Job, error) {
	for _, job := range jobs {
		if len(job.ShouldBeLockedBefore()) > 0 {
			return nil, fmt.Errorf("director job '%s' specifies locking dependencies, which are not allowed for director jobs", job.Name())
		}
	}
	return jobs, nil
}
