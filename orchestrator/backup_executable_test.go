package orchestrator_test

import (
	"fmt"

	"github.com/cloudfoundry/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BackupExecutables", func() {
	var (
		err        error
		executable executor.Executable
		fakeJob    *fakes.FakeJob
	)

	BeforeEach(func() {
		fakeJob = new(fakes.FakeJob)
	})

	Context("NewBackupExecutable", func() {
		BeforeEach(func() {
			executable = orchestrator.NewBackupExecutable(fakeJob)
		})
		JustBeforeEach(func() {
			err = executable.Execute()
		})

		It("executes backup", func() {
			Expect(fakeJob.BackupCallCount()).To(Equal(1))
		})

		Context("when the backup fails", func() {
			BeforeEach(func() {
				fakeJob.BackupReturns(fmt.Errorf("I failed at backup"))
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(ContainSubstring("I failed at backup")))
			})
		})
	})
})
