package all_deployments_executor

func NewSerialDeploymentExecutor() SerialDeploymentExecutor {
	return SerialDeploymentExecutor{}
}

type SerialDeploymentExecutor struct {
}

func (s SerialDeploymentExecutor) Run(executables []Executable) []DeploymentError {
	var errors []DeploymentError

	for _, executable := range executables {
		if err := executable.Execute(); err.Errs != nil {
			errors = append(errors, err)
		}
	}
	return errors
}
