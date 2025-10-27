package orchestrator_test

import (
	"fmt"

	"github.com/cloudfoundry/bosh-backup-and-restore/orchestrator"

	"errors"

	goerr "github.com/pkg/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type ErrorCase struct {
	name             string
	errors           []error
	expectedExitCode int
}

var _ = Describe("Error", func() {
	var genericError = goerr.Wrap(errors.New("just a little error"), "generic cause")
	var lockError = orchestrator.NewLockError("LOCK_ERROR")
	var backupError = orchestrator.NewBackupError("BACKUP_ERROR")
	var postBackupUnlockError = orchestrator.NewPostUnlockError("POST_BACKUP_ERROR")
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

	Describe("BuildExitCode", func() {
		Context("when there are no errors", func() {
			It("returns exit code 0", func() {
				exitCode := orchestrator.BuildExitCode([]error{})
				Expect(exitCode).To(Equal(0))
			})
		})

		Context("errors", func() {
			errorCases := []ErrorCase{
				{"genericError", []error{genericError}, 1},
				{"backupError", []error{backupError}, 1},
				{"lockError", []error{lockError}, 4},
				{"unlockError", []error{postBackupUnlockError}, 8},
				{"cleanupError", []error{cleanupError}, 16},
			}

			for i := range errorCases {
				errorCase := errorCases[i]

				It(fmt.Sprintf("returns exit code %v in case of %v", errorCase.expectedExitCode, errorCase.name), func() {
					actualExitCode := orchestrator.BuildExitCode(errorCase.errors)
					Expect(actualExitCode).To(Equal(errorCase.expectedExitCode))
				})
			}
		})

		Context("when there is only a lock error", func() {
			var exitCode int

			BeforeEach(func() {
				exitCode = orchestrator.BuildExitCode([]error{lockError})
			})

			It("returns exit code 4", func() {
				Expect(exitCode).To(Equal(4))
			})
		})

		Context("when there is a backup error and a cleanup error", func() {
			It("returns exit code 17 (16 | 1)", func() {
				exitCode := orchestrator.BuildExitCode([]error{cleanupError, backupError})
				Expect(exitCode).To(Equal(17))
			})
		})

		Context("when there is a generic error and a cleanup error", func() {
			It("returns exit code 17 (16 | 1)", func() {
				exitCode := orchestrator.BuildExitCode([]error{cleanupError, genericError})
				Expect(exitCode).To(Equal(17))
			})
		})

		Context("when there are two errors of the same type", func() {
			It("the error bit is only set once", func() {
				exitCode := orchestrator.BuildExitCode([]error{cleanupError, cleanupError})
				Expect(exitCode).To(Equal(16))
			})
		})
	})

	Describe("ConvertErrors", func() {
		var errorOne = errors.New("error one")
		var errorTwo = errors.New("error two")
		var errorThree = errors.New("error three")
		var errs = []error{errorOne, errorTwo, errorThree}

		It("converts a list of errors into an Error", func() {
			Expect(orchestrator.ConvertErrors(errs)).To(Equal(orchestrator.Error(errs)))
		})

		Context("when the errors list is empty", func() {
			It("returns nil", func() {
				Expect(orchestrator.ConvertErrors([]error{}) == nil).To(BeTrue())
			})
		})

		Context("when the errors list is nil", func() {
			It("returns nil", func() {
				Expect(orchestrator.ConvertErrors(nil) == nil).To(BeTrue())
			})
		})

		Context("when the errors list contains Errors", func() {
			It("flattens them", func() {
				Expect(orchestrator.ConvertErrors([]error{orchestrator.Error(errs)})).To(Equal(orchestrator.Error(errs)))
			})
		})

		Context("when the errors list contains nested Errors", func() {
			It("flattens them", func() {
				Expect(orchestrator.ConvertErrors([]error{orchestrator.Error([]error{orchestrator.Error(errs)})})).To(Equal(orchestrator.Error(errs)))
			})

			Context("when the resulting list is empty", func() {
				It("returns nil", func() {
					Expect(orchestrator.ConvertErrors([]error{orchestrator.Error([]error{orchestrator.Error([]error{})})}) == nil).To(BeTrue())
				})
			})
		})
	})
})
