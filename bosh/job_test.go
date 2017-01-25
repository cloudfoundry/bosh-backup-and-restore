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
		jobScripts = bosh.BackupAndRestoreScripts{
			"/var/vcap/jobs/foo/bin/p-restore",
			"/var/vcap/jobs/foo/bin/p-backup",
			"/var/vcap/jobs/foo/bin/p-pre-backup-lock",
			"/var/vcap/jobs/foo/bin/p-post-backup-unlock",
		}
	})

	JustBeforeEach(func() {
		job = bosh.NewJob(jobScripts)
	})
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

	Describe("PreBackupScript", func() {
		It("returns the pre-backup script", func() {
			Expect(job.PreBackupScript()).To(Equal(bosh.Script("/var/vcap/jobs/foo/bin/p-pre-backup-lock")))
		})
		Context("no pre-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns nil", func() {
				Expect(job.PreBackupScript()).To(BeEmpty())
			})
		})
	})

	Describe("HasPreBackup", func() {
		It("returns true", func() {
			Expect(job.HasPreBackup()).To(BeTrue())
		})

		Context("no pre-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns false", func() {
				Expect(job.HasPreBackup()).To(BeFalse())
			})
		})
	})

	Describe("PostBackupScript", func() {
		It("returns the post-backup script", func() {
			Expect(job.PostBackupScript()).To(Equal(bosh.Script("/var/vcap/jobs/foo/bin/p-post-backup-unlock")))
		})
		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns nil", func() {
				Expect(job.PostBackupScript()).To(BeEmpty())
			})
		})
	})

	Describe("HasPostBackup", func() {
		It("returns true", func() {
			Expect(job.HasPostBackup()).To(BeTrue())
		})

		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns false", func() {
				Expect(job.HasPostBackup()).To(BeFalse())
			})
		})
	})

	Describe("RestoreScript", func() {
		It("returns the post-backup script", func() {
			Expect(job.RestoreScript()).To(Equal(bosh.Script("/var/vcap/jobs/foo/bin/p-restore")))
		})
		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-backup"}
			})
			It("returns nil", func() {
				Expect(job.RestoreScript()).To(BeEmpty())
			})
		})
	})

	Describe("HasRestore", func() {
		It("returns true", func() {
			Expect(job.HasRestore()).To(BeTrue())
		})

		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-backup"}
			})
			It("returns false", func() {
				Expect(job.HasRestore()).To(BeFalse())
			})
		})
	})
})
