package orchestrator_test

import (
	"github.com/pivotal-cf/pcf-backup-and-restore/orchestrator"

	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error", func() {
	var cleanupError = orchestrator.CleanupError{}
	var genericError = errors.New("You are fake news")
	var postBackupUnlockError = orchestrator.PostBackupUnlockError{}

	Describe("IsCleanup", func() {
		It("returns true when there is only one error - a cleanup error", func() {
			errors := orchestrator.Error{cleanupError}
			Expect(errors.IsCleanup()).To(BeTrue())
		})

		It("returns false when there is only one error - not a cleanup error", func() {
			errors := orchestrator.Error{genericError}
			Expect(errors.IsCleanup()).To(BeFalse())
		})

		It("returns false when empty", func() {
			var errors orchestrator.Error
			Expect(errors.IsCleanup()).To(BeFalse())
		})

		It("returns false when there is more than one error - with a cleanup error", func() {
			errors := orchestrator.Error{genericError, cleanupError}
			Expect(errors.IsCleanup()).To(BeFalse())
		})

		It("returns false when there is a cleanup error and a post backup error", func() {
			errors := orchestrator.Error{postBackupUnlockError, cleanupError}
			Expect(errors.IsCleanup()).To(BeFalse())
		})
	})

	Describe("IsPostBackup", func() {
		It("returns false when empty", func() {
			var errors orchestrator.Error
			Expect(errors.IsPostBackup()).To(BeFalse())
		})

		It("returns true when there is only one error - a post-backup-unlock error", func() {
			errors := orchestrator.Error{postBackupUnlockError}
			Expect(errors.IsPostBackup()).To(BeTrue())
		})

		It("returns true when there are many errors and one of the is a post-backup-unlock error", func() {
			errors := orchestrator.Error{postBackupUnlockError, cleanupError}
			Expect(errors.IsPostBackup()).To(BeTrue())
		})

		It("returns false when there are many errors and any of them is a generic error", func() {
			errors := orchestrator.Error{postBackupUnlockError, genericError}
			Expect(errors.IsPostBackup()).To(BeFalse())
		})
	})

	Describe("IsFatal", func() {
		It("returns true when there is one error - a generic error", func() {
			errors := orchestrator.Error{genericError}
			Expect(errors.IsFatal()).To(BeTrue())
		})

		It("returns false when there are no errors", func() {
			var errors orchestrator.Error
			Expect(errors.IsFatal()).To(BeFalse())
		})

		It("returns true when there are many errors and any of them is a generic error", func() {
			errors := orchestrator.Error{postBackupUnlockError, genericError}
			Expect(errors.IsFatal()).To(BeTrue())
		})

		It("returns false when there are many errors but none of them is a generic error", func() {
			errors := orchestrator.Error{postBackupUnlockError, cleanupError}
			Expect(errors.IsFatal()).To(BeFalse())
		})
	})

	Describe("IsNil", func() {
		It("returns false when there are any errors", func() {
			errors := orchestrator.Error{genericError}
			Expect(errors.IsNil()).To(BeFalse())
		})

		It("returns true when empty", func() {
			errors := orchestrator.Error{}
			Expect(errors.IsNil()).To(BeTrue())
		})
		It("returns true when nil", func() {
			var errors orchestrator.Error
			Expect(errors.IsNil()).To(BeTrue())
		})

	})
})
