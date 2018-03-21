package executor

//go:generate counterfeiter -o fakes/fake_executor.go . Executor
type Executor interface {
	Run([][]Executable) []error
}

type Executable interface {
	Execute() error
}
