package orderer

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PreBackupLockConstraintSpecifier", func() {
	It("returns the job specifier for pre backup lock", func() {
		backupLockBeforeSpecifier := []orchestrator.JobSpecifier{{Name: "name1", Release: "release1"}}

		fakeJob := new(fakes.FakeJob)
		fakeJob.BackupShouldBeLockedBeforeReturns(backupLockBeforeSpecifier)

		Expect(NewBackupOrderConstraintSpecifier().Before(fakeJob)).To(Equal(backupLockBeforeSpecifier))
	})
})
