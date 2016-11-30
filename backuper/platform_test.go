package backuper_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"
)

var _ = Describe("Platform", func() {
	var platform backuper.Platform
	var deployments []backuper.Deployment
	var deployment1 *fakes.FakeDeployment
	var deployment2 *fakes.FakeDeployment
	BeforeEach(func() {
		deployment1 = new(fakes.FakeDeployment)
		deployment2 = new(fakes.FakeDeployment)
	})
	JustBeforeEach(func() {
		platform = backuper.NewBoshPlatform(deployments)
	})

	Context("Backup", func() {
		var backupError error
		JustBeforeEach(func() {
			backupError = platform.Backup()
		})

		Context("single deployment", func() {
			BeforeEach(func() {
				deployments = []backuper.Deployment{deployment1}
			})

			It("invokes the backup the deployment", func() {
				Expect(deployment1.BackupCallCount()).To(Equal(1))
			})
			Context("failure", func() {
				var actualError = fmt.Errorf("we need global warming!")
				BeforeEach(func() {
					deployment1.BackupReturns(actualError)
				})
				It("returns a error if backing up any deployment fails", func() {
					Expect(backupError).To(MatchError(actualError))
				})
			})
		})

		Context("multiple deployment", func() {
			BeforeEach(func() {
				deployments = []backuper.Deployment{deployment1, deployment2}
			})
			It("invokes the backup each deployment", func() {
				Expect(deployment1.BackupCallCount()).To(Equal(1))
				Expect(deployment2.BackupCallCount()).To(Equal(1))
			})
			Context("failure", func() {
				var actualError = fmt.Errorf("we will have the best laws")
				BeforeEach(func() {
					deployment2.BackupReturns(actualError)
				})
				It("returns a error if backing up any deployment fails", func() {
					Expect(backupError).To(MatchError(actualError))
				})
			})
		})
	})

	Context("Cleanup", func() {
		var cleanupError error
		JustBeforeEach(func() {
			cleanupError = platform.Cleanup()
		})

		Context("single deployment", func() {
			BeforeEach(func() {
				deployments = []backuper.Deployment{deployment1}
			})

			It("invokes the cleanup of the deployment", func() {
				Expect(deployment1.CleanupCallCount()).To(Equal(1))
			})
			Context("failure", func() {
				var actualError = fmt.Errorf("I beat china all the time")
				BeforeEach(func() {
					deployment1.CleanupReturns(actualError)
				})
				It("returns a error if cleaning up any deployment fails", func() {
					Expect(cleanupError).To(MatchError(actualError))
				})
			})
		})

		Context("multiple deployment", func() {
			BeforeEach(func() {
				deployments = []backuper.Deployment{deployment1, deployment2}
			})
			It("invokes the cleanup of each deployment", func() {
				Expect(deployment1.CleanupCallCount()).To(Equal(1))
				Expect(deployment2.CleanupCallCount()).To(Equal(1))
			})
			Context("failure", func() {
				var actualError = fmt.Errorf("we will have the best laws")
				BeforeEach(func() {
					deployment2.CleanupReturns(actualError)
				})
				It("returns a error if cleaning up any deployment fails", func() {
					Expect(cleanupError).To(MatchError(actualError))
				})
			})
		})
	})
})
