package deployment

func NewSerialExecutor() SerialExecutor { //nolint:unused
	return SerialExecutor{}
}

type SerialExecutor struct {
}

func (s SerialExecutor) Run(executables []Executable) []DeploymentError {
	var errors []DeploymentError

	for _, executable := range executables {
		if err := executable.Execute(); err.Errs != nil {
			errors = append(errors, err)
		}
	}
	return errors
}
