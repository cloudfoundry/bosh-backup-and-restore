package orchestrator_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BackupExecutables", func() {
	var (
		fakeInstance *fakes.FakeInstance
		err          error
		executable   executor.Executable
	)

	BeforeEach(func() {
		fakeInstance = new(fakes.FakeInstance)
	})

	Context("NewBackupExecutable", func() {
		BeforeEach(func() {
			executable = orchestrator.NewBackupExecutable(fakeInstance)
		})
		JustBeforeEach(func() {
			err = executable.Execute()
		})

		It("executes backup", func() {
			Expect(fakeInstance.BackupCallCount()).To(Equal(1))
		})

		Context("when the backup fails", func() {
			BeforeEach(func() {
				fakeInstance.BackupReturns(fmt.Errorf("I failed at backup"))
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(ContainSubstring("I failed at backup")))
			})
		})
	})
})
