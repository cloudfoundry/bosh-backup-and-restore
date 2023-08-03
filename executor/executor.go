package executor

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_executor.go . Executor
type Executor interface {
	Run([][]Executable) []error
}

//counterfeiter:generate -o fakes/fake_executable.go . Executable
type Executable interface {
	Execute() error
}
