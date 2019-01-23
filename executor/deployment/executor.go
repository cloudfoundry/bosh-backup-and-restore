package deployment

type DeploymentExecutor interface {
	Run([]Executable) []DeploymentError
}

type Executable interface {
	Execute() DeploymentError
}
