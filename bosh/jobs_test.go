package bosh_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
)

var _ = Describe("Jobs", func() {
	var jobs bosh.Jobs
	var scripts bosh.BackupAndRestoreScripts
	JustBeforeEach(func() {
		jobs = bosh.NewJobs(scripts)
	})

	Describe("NewJobs", func() {
		Context("when there are two jobs each with a backup script", func() {
			BeforeEach(func() {
				scripts = bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/p-backup",
					"/var/vcap/jobs/bar/bin/p-backup",
				}
			})
			It("groups scripts to create jobs", func() {
				Expect(jobs).To(ConsistOf(
					bosh.NewJob(bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-backup"}),
					bosh.NewJob(bosh.BackupAndRestoreScripts{"/var/vcap/jobs/bar/bin/p-backup"}),
				))
			})
		})

		Context("when there is one job with a backup script", func() {
			BeforeEach(func() {
				scripts = bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/p-backup",
				}
			})
			It("groups scripts to create jobs", func() {
				Expect(jobs).To(ConsistOf(
					bosh.NewJob(bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-backup"}),
				))
			})
		})
	})

	Context("contains jobs with backup script",func() {
		BeforeEach(func() {
			scripts = bosh.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/p-backup",
				"/var/vcap/jobs/bar/bin/p-restore",
			}
		})

		Describe("Backupable", func() {
			It("returns the backupable job", func() {
				Expect(jobs.Backupable()).To(ConsistOf(bosh.NewJob(bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-backup"}), ))
			})
		})

		Describe("AnyAreBackupable", func() {
			It("returns true", func() {
				Expect(jobs.AnyAreBackupable()).To(BeTrue())
			})
		})
	})

	Context("contains no jobs with backup script",func(){
		BeforeEach(func() {
			scripts = bosh.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/p-restore",
			}
		})

		Describe("Backupable", func() {
			It("returns empty",func(){
				Expect(jobs.Backupable()).To(BeEmpty())
			})
		})

		Describe("AnyAreBackupable", func() {
			It("returns false", func() {
				Expect(jobs.AnyAreBackupable()).To(BeFalse())
			})
		})
	})



	Context("contains jobs with pre-backup-lock scripts",func(){
		BeforeEach(func() {
			scripts = bosh.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/p-pre-backup-lock",
				"/var/vcap/jobs/foo/bin/p-backup",
				"/var/vcap/jobs/bar/bin/p-restore",
			}
		})

		Describe("PreBackupable", func() {
			It("returns the lockable job",func(){
				Expect(jobs.PreBackupable()).To(ConsistOf(bosh.NewJob(
					bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-pre-backup-lock","/var/vcap/jobs/foo/bin/p-backup"}),
				))
			})
		})
		Context("contains no jobs with backup script",func(){
			BeforeEach(func() {
				scripts = bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/bar/bin/p-restore",
				}
			})

			It("returns empty",func(){
				Expect(jobs.PreBackupable()).To(BeEmpty())
			})
		})
	})

	Describe("PostBackupable", func() {
		Context("contains jobs with pre-backup-lock scripts",func(){
			BeforeEach(func() {
				scripts = bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/p-backup",
					"/var/vcap/jobs/foo/bin/p-post-backup-unlock",
					"/var/vcap/jobs/bar/bin/p-restore",
				}
			})

			It("returns the unlockable job",func(){
				Expect(jobs.PostBackupable()).To(ConsistOf(bosh.NewJob(
					bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-post-backup-unlock","/var/vcap/jobs/foo/bin/p-backup"}),
				))
			})
		})
		Context("contains no jobs with backup script",func(){
			BeforeEach(func() {
				scripts = bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/bar/bin/p-restore",
				}
			})

			It("returns empty",func(){
				Expect(jobs.PostBackupable()).To(BeEmpty())
			})
		})
	})

	Describe("Restorable", func() {
		Context("contains jobs with restore scripts",func(){
			BeforeEach(func() {
				scripts = bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/p-backup",
					"/var/vcap/jobs/foo/bin/p-post-backup-unlock",
					"/var/vcap/jobs/bar/bin/p-restore",
				}
			})

			It("returns the unlockable job",func(){
				Expect(jobs.Restorable()).To(ConsistOf(bosh.NewJob(
					bosh.BackupAndRestoreScripts{"/var/vcap/jobs/bar/bin/p-restore"}),
				))
			})
		})
		Context("contains no jobs with backup script",func(){
			BeforeEach(func() {
				scripts = bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/bar/bin/p-backup",
				}
			})

			It("returns empty",func(){
				Expect(jobs.Restorable()).To(BeEmpty())
			})
		})
	})
})
