package executor

type Executor interface {
	Run([][]Executable) []error
}

type Executable interface {
	Execute() error
}
