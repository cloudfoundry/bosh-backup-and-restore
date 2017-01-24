package bosh_test

import (
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Job", func() {
	var job bosh.Job
	var jobScripts bosh.BackupAndRestoreScripts

	BeforeEach(func() {
		jobScripts = bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-backup"}
	})

	JustBeforeEach(func() {
		job = bosh.NewJob(jobScripts)
	})
	Context("when job script is just the backup script", func() {
		Describe("ArtifactDirectory", func() {
			It("calculates the artifact directory based on the name", func() {
				Expect(job.ArtifactDirectory()).To(Equal("/var/vcap/store/backup/foo"))
			})
		})
		Describe("BackupScript", func() {
			It("returns the backup script", func() {
				Expect(job.BackupScript()).To(Equal(bosh.Script("/var/vcap/jobs/foo/bin/p-backup")))
			})
			Context("no backup scripts exist", func() {
				BeforeEach(func() {
					jobScripts = bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
				})
				It("returns nil", func() {
					Expect(job.BackupScript()).To(BeEmpty())
				})
			})
		})
		Describe("HasBackup", func() {
			It("returns true", func() {
				Expect(job.HasBackup()).To(BeTrue())
			})

			Context("no backup scripts exist", func() {
				BeforeEach(func() {
					jobScripts = bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
				})
				It("returns false", func() {
					Expect(job.HasBackup()).To(BeFalse())
				})
			})
		})
	})
})
