package executor

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

func NewSerialExecutor() SerialExecutor {
	return SerialExecutor{}
}

type SerialExecutor struct {
}

func (s SerialExecutor) Run(executablesList [][]orchestrator.Executable) []error {
	var errors []error
	for _, executables := range executablesList {
		for _, executable := range executables {
			if err := executable.Execute(); err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
