package orderer

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PreRestoreLockConstraintSpecifier", func() {
	It("returns the job specifier for pre restore lock", func() {
		restoreLockBeforeSpecifier := []orchestrator.JobSpecifier{{Name: "name1", Release: "release1"}}

		fakeJob := new(fakes.FakeJob)
		fakeJob.RestoreShouldBeLockedBeforeReturns(restoreLockBeforeSpecifier)

		Expect(NewRestoreOrderConstraintSpecifier().Before(fakeJob)).To(Equal(restoreLockBeforeSpecifier))
	})
})
