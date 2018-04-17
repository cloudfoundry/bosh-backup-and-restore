package executor

func NewParallelExecutor() ParallelExecutor {
	return ParallelExecutor{
		maxInFlight: 10,
	}
}

type ParallelExecutor struct {
	maxInFlight int
}

func (s ParallelExecutor) Run(executablesList [][]Executable) []error {
	var errors []error
	for _, executables := range executablesList {
		guard := make(chan bool, s.maxInFlight)
		errs := make(chan error, len(executables))

		for _, executable := range executables {
			guard <- true
			go func(executable Executable) {
				errs <- executable.Execute()
				<-guard
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
