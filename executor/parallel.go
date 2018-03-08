package executor

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

func NewParallelExecutor() ParallelExecutor {
	return ParallelExecutor{}
}

type ParallelExecutor struct {
}

func (s ParallelExecutor) Run(executablesList [][]orchestrator.Executable) []error {
	var errors []error
	for _, executables := range executablesList {
		errs := make(chan error, len(executables))

		for _, executable := range executables {
			go func(executable orchestrator.Executable) {
				errs <- executable.Execute()
			}(executable)
		}

		for range executables {
			err := <-errs
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
