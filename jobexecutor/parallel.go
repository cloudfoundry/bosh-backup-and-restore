package jobexecutor

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"sync"
)

func NewParallelJobExecutor() ParallelJobExecutor {
	return ParallelJobExecutor{}
}

type ParallelJobExecutor struct {
}

func (s ParallelJobExecutor) Run(runMethod func(orchestrator.Job) error, jobs [][]orchestrator.Job) []error {
	var errors []error
	for _, jobList := range jobs {
		var wg sync.WaitGroup
		errs := make(chan error, len(jobList))

		for _, job := range jobList {
			wg.Add(1)
			go func(j orchestrator.Job) {
				defer wg.Done()
				err := runMethod(j)
				errs <- err
			}(job)
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
