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

	Describe("Backupable", func() {
		Context("contains jobs with backup script",func(){
			BeforeEach(func() {
				scripts = bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/p-backup",
					"/var/vcap/jobs/bar/bin/p-restore",
				}
			})
			It("returns the backupable job",func(){
				Expect(jobs.Backupable()).To(ConsistOf(bosh.NewJob(bosh.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-backup"}),))
			})
		})
		Context("contains no jobs with backup script",func(){
			BeforeEach(func() {
				scripts = bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/bar/bin/p-restore",
				}
			})
			It("returns empty",func(){
				Expect(jobs.Backupable()).To(BeEmpty())
			})
		})
	})
})
