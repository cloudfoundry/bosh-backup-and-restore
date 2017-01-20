package bosh_test

import (
	. "github.com/pivotal-cf/pcf-backup-and-restore/bosh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup and Restore Scripts", func() {
	Describe("NewBackupAndRestoreScripts", func() {
		Context("Backup", func() {
			It("returns the matching script when it has only one backup script", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/p-backup",
										  "/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/p-backup"}))
			})

			It("returns empty when backup scripts is in a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/foo/p-backup",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})

			It("returns empty when backup scripts in bin with a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/foo/bin/p-backup",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
			It("returns the matching scripts when there are multiple backup scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/p-backup",
										  "/var/vcap/jobs/consul_agent/bin/p-backup",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/p-backup",
					"/var/vcap/jobs/consul_agent/bin/p-backup",
				}))
			})

			It("returns empty when there are backup scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
		})

		Context("Restore", func() {
			It("returns the matching script when it has only one restore script", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/p-restore",
										  "/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/p-restore"}))
			})

			It("returns empty when restore scripts is in a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/foo/p-restore",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})

			It("returns empty when restore scripts in bin with a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/foo/bin/p-restore",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
			It("returns the matching scripts when there are multiple restore scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/p-restore",
										  "/var/vcap/jobs/consul_agent/bin/p-restore",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/p-restore",
					"/var/vcap/jobs/consul_agent/bin/p-restore",
				}))
			})

			It("returns empty when there are restore scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										  "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										  "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
		})
	})

	Describe("BackupOnly", func() {
		It("returns the p-backup scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										 "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										 "/var/vcap/jobs/cloud_controller_clock/bin/p-backup",
										 "/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
										 "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.BackupOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/p-backup"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										 "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										 "/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
										 "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.BackupOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all p-backup scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/p-backup",
										 "/var/vcap/jobs/cloud_controller/bin/p-backup",
										 "/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
										 "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.BackupOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/p-backup",
																	"/var/vcap/jobs/cloud_controller/bin/p-backup",
			}))
		})
	})

	Describe("RestoreOnly", func() {
		It("returns the p-backup scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										 "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										 "/var/vcap/jobs/cloud_controller_clock/bin/p-restore",
										 "/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
										 "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.RestoreOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/p-restore"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
										 "/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
										 "/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
										 "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.RestoreOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all p-backup scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/p-restore",
										 "/var/vcap/jobs/cloud_controller/bin/p-restore",
										 "/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
										 "/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.RestoreOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/p-restore",
																	"/var/vcap/jobs/cloud_controller/bin/p-restore",
			}))
		})
	})
})

var _ = Describe("Script", func() {
	var (
		script Script
		result string
		err error
	)

	JustBeforeEach(func() {
		result, err = script.JobName()
	})

	Describe("JobName", func() {
		BeforeEach(func() {
			script = Script("/var/vcap/jobs/a-job-name/p-backup")
		})

		It("returns the job name for a given bosh job script", func() {
			Expect(result).To(Equal("a-job-name"))
		})

		Context("when provided script is not a job script", func() {
			BeforeEach(func() {
				script = Script("/var/vcap/packages/job/some-script")
			})

			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
