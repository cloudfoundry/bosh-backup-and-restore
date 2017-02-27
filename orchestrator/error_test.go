package orchestrator_test

import (
	"github.com/pivotal-cf/pcf-backup-and-restore/orchestrator"

	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error", func() {
	var genericError = errors.New("You are fake news")

	var lockError = orchestrator.NewLockError("LOCK_ERROR")
	var backupError = orchestrator.NewBackupError("BACKUP_ERROR")
	var postBackupUnlockError = orchestrator.NewPostBackupUnlockError("POST_BACKUP_ERROR")
	var cleanupError = orchestrator.NewCleanupError("CLEANUP_ERROR")

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

	Describe("ProcessBackupError", func() {
		Context("when there are no errors", func() {
			It("returns exit code 0", func() {
				exitCode, errorMessage := orchestrator.ProcessBackupError([]error{})
				Expect(exitCode).To(Equal(0))
				Expect(errorMessage).To(Equal(""))
			})
		})

		Context("when there is only a lock error", func() {
			var exitCode int
			var errorMessage string

			BeforeEach(func() {
				exitCode, errorMessage = orchestrator.ProcessBackupError([]error{lockError})
			})

			It("returns exit code 4", func() {
				Expect(exitCode).To(Equal(4))
				Expect(errorMessage).To(ContainSubstring("LOCK_ERROR"))
			})

			It("only reports one error", func() {
				Expect(errorMessage).To(ContainSubstring("1 error occurred:"))
			})
		})

		Context("when there is only a backup error", func() {
			It("returns exit code 8", func() {
				exitCode, errorMessage := orchestrator.ProcessBackupError([]error{backupError})
				Expect(exitCode).To(Equal(8))
				Expect(errorMessage).To(ContainSubstring("BACKUP_ERROR"))
			})
		})

		Context("when there is only an unlock error", func() {
			It("returns exit code 16", func() {
				exitCode, errorMessage := orchestrator.ProcessBackupError([]error{postBackupUnlockError})
				Expect(exitCode).To(Equal(16))
				Expect(errorMessage).To(ContainSubstring("POST_BACKUP_ERROR"))
			})
		})

		Context("when there is only a cleanup error", func() {
			It("returns exit code 32", func() {
				exitCode, errorMessage := orchestrator.ProcessBackupError([]error{cleanupError})
				Expect(exitCode).To(Equal(32))
				Expect(errorMessage).To(ContainSubstring("CLEANUP_ERROR"))
			})
		})

		Context("when there is a lock error and a cleanup error", func() {
			It("returns exit code 36 (32 | 4)", func() {
				exitCode, errorMessage := orchestrator.ProcessBackupError([]error{lockError, cleanupError})
				Expect(exitCode).To(Equal(36))
				Expect(errorMessage).To(ContainSubstring("LOCK_ERROR"))
				Expect(errorMessage).To(ContainSubstring("CLEANUP_ERROR"))
			})
		})

		Context("when there are two errors of the same type", func() {
			It("the error bit is only set once", func() {
				exitCode, errorMessage := orchestrator.ProcessBackupError([]error{cleanupError, cleanupError})
				Expect(exitCode).To(Equal(32))
				Expect(errorMessage).To(ContainSubstring("2 errors occurred:"))
				Expect(errorMessage).To(ContainSubstring("CLEANUP_ERROR"))
			})
		})

		Context("when there is a generic error", func() {
			It("returns exit code 1", func() {
				exitCode, errorMessage := orchestrator.ProcessBackupError([]error{errors.New("FAIL")})
				Expect(exitCode).To(Equal(1))
				Expect(errorMessage).To(Equal("FAIL"))
			})
		})
	})
})
