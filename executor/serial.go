package executor

func NewSerialExecutor() SerialExecutor {
	return SerialExecutor{}
}

type SerialExecutor struct {
}

func (s SerialExecutor) Run(executablesList [][]Executable) []error {
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
