package executor

//go:generate counterfeiter -o fakes/fake_executor.go . Executor
type Executor interface {
	Run([][]Executable) []error
}

//go:generate counterfeiter -o fakes/fake_executable.go . Executable
type Executable interface {
	Execute() error
}
