package executor

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

func NewParallelJobExecutor() ParallelJobExecutor {
	return ParallelJobExecutor{}
}

type ParallelJobExecutor struct {
}

func (s ParallelJobExecutor) Run(runMethod func(orchestrator.Job) error, jobs [][]orchestrator.Job) []error {
	var errors []error
	for _, jobList := range jobs {
		errs := make(chan error, len(jobList))

		for _, job := range jobList {
			go func(j orchestrator.Job) {
				errs <- runMethod(j)
			}(job)
		}

		for range jobList {
			err := <- errs
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
