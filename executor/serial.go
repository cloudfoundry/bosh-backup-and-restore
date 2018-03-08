package executor

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

func NewSerialJobExecutor() SerialJobExecutor {
	return SerialJobExecutor{}
}

type SerialJobExecutor struct {
}

func (s SerialJobExecutor) Run(runMethod func(job orchestrator.Job) error, jobs [][]orchestrator.Job) []error {
	var errors []error
	for _, jobList := range jobs {
		for _, job := range jobList {
			if err := runMethod(job); err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
