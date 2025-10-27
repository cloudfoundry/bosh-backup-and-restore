package executor_test

import (
	. "github.com/cloudfoundry/bosh-backup-and-restore/executor"

	"github.com/cloudfoundry/bosh-backup-and-restore/executor/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Executor", func() {
	ExecutorTests := func(name string, executor Executor) {
		Describe(name, func() {
			var errs []error
			var executable1, executable2, executable3, executable4 *fakes.FakeExecutable
			var orderOfExecution []string

			BeforeEach(func() {
				executable1 = new(fakes.FakeExecutable)
				executable1.ExecuteStub = func() error {
					orderOfExecution = append(orderOfExecution, "executable1")
					return nil
				}

				executable2 = new(fakes.FakeExecutable)
				executable2.ExecuteStub = func() error {
					orderOfExecution = append(orderOfExecution, "executable2")
					return nil
				}

				executable3 = new(fakes.FakeExecutable)
				executable3.ExecuteStub = func() error {
					orderOfExecution = append(orderOfExecution, "executable3")
					return nil
				}

				executable4 = new(fakes.FakeExecutable)
				executable4.ExecuteStub = func() error {
					orderOfExecution = append(orderOfExecution, "executable4")
					return nil
				}
			})

			JustBeforeEach(func() {
				errs = executor.Run([][]Executable{
					{executable1},
					{executable2, executable3},
					{executable4},
				})
			})

			It("executes each batch in parallel", func() {
				Expect(errs).To(HaveLen(0))

				Expect(executable1.ExecuteCallCount()).To(Equal(1))
				Expect(executable2.ExecuteCallCount()).To(Equal(1))
				Expect(executable3.ExecuteCallCount()).To(Equal(1))
				Expect(executable4.ExecuteCallCount()).To(Equal(1))

				Expect(orderOfExecution[0]).To(Equal("executable1"))
				Expect(orderOfExecution[1:3]).To(ConsistOf("executable2", "executable3"))
				Expect(orderOfExecution[3]).To(Equal("executable4"))
			})

			Context("when some executables fail", func() {
				BeforeEach(func() {
					executable2.ExecuteReturns(errors.New("error from executable2"))
					executable4.ExecuteReturns(errors.New("error from executable4"))
				})

				It("still executes all the executables and returns the list of errors", func() {
					Expect(errs).To(ConsistOf(
						MatchError("error from executable2"),
						MatchError("error from executable4"),
					))

					Expect(executable1.ExecuteCallCount()).To(Equal(1))
					Expect(executable2.ExecuteCallCount()).To(Equal(1))
					Expect(executable3.ExecuteCallCount()).To(Equal(1))
					Expect(executable4.ExecuteCallCount()).To(Equal(1))
				})
			})
		})
	}

	ExecutorTests("SerialExecutor", NewSerialExecutor())
	ExecutorTests("ParallelExecutor", NewParallelExecutor())
})
