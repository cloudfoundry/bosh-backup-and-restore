package instance_test

import (
	. "github.com/pivotal-cf/bosh-backup-and-restore/instance"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup and Restore Scripts", func() {
	Describe("NewBackupAndRestoreScripts", func() {
		Context("Backup", func() {
			It("returns the matching script when it has only one backup script", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/b-backup",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-backup"}))
			})

			It("returns empty when backup scripts is in a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/b-backup",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})

			It("returns empty when backup scripts in bin with a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/bin/b-backup",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
			It("returns the matching scripts when there are multiple backup scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/b-backup",
					"/var/vcap/jobs/consul_agent/bin/b-backup",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/b-backup",
					"/var/vcap/jobs/consul_agent/bin/b-backup",
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
					"/var/vcap/jobs/cloud_controller_clock/bin/b-restore",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-restore"}))
			})

			It("returns empty when restore scripts is in a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/b-restore",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})

			It("returns empty when restore scripts in bin with a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/bin/b-restore",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
			It("returns the matching scripts when there are multiple restore scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/b-restore",
					"/var/vcap/jobs/consul_agent/bin/b-restore",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/b-restore",
					"/var/vcap/jobs/consul_agent/bin/b-restore",
				}))
			})

			It("returns empty when there are no restore scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
		})

		Context("PreBackupLock", func() {
			It("returns the matching scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/b-pre-backup-lock",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/b-pre-backup-lock",
				}))
			})
		})

		Context("PostBackupUnlock", func() {
			It("returns the matching scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/b-post-backup-unlock",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/b-post-backup-unlock",
				}))
			})
		})

		Context("Metadata", func() {
			It("returns the matching scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/b-metadata",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/b-metadata",
				}))
			})
		})
	})

	Describe("BackupOnly", func() {
		It("returns the b-backup scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/b-backup",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.BackupOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-backup"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.BackupOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all b-backup scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-backup",
				"/var/vcap/jobs/cloud_controller/bin/b-backup",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.BackupOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-backup",
				"/var/vcap/jobs/cloud_controller/bin/b-backup",
			}))
		})
	})

	Describe("RestoreOnly", func() {
		It("returns the b-backup scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/b-restore",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.RestoreOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-restore"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.RestoreOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all b-backup scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-restore",
				"/var/vcap/jobs/cloud_controller/bin/b-restore",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.RestoreOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-restore",
				"/var/vcap/jobs/cloud_controller/bin/b-restore",
			}))
		})
	})

	Describe("MetadataOnly", func() {
		It("returns the b-backup scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/b-metadata",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.MetadataOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-metadata"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.MetadataOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all b-backup scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-metadata",
				"/var/vcap/jobs/cloud_controller/bin/b-metadata",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.MetadataOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-metadata",
				"/var/vcap/jobs/cloud_controller/bin/b-metadata",
			}))
		})
	})

	Describe("PreBackupLockOnly", func() {
		It("returns the b-pre-backup-lock scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/b-backup",
				"/var/vcap/jobs/cloud_controller_clock/bin/b-pre-backup-lock",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PreBackupLockOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-pre-backup-lock"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PreBackupLockOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all b-pre-backup-lock scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-pre-backup-lock",
				"/var/vcap/jobs/cloud_controller/bin/b-pre-backup-lock",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PreBackupLockOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-pre-backup-lock",
				"/var/vcap/jobs/cloud_controller/bin/b-pre-backup-lock",
			}))
		})
	})

	Describe("PostBackupUnlockOnly", func() {
		It("returns the b-post-backup-unlock scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/b-backup",
				"/var/vcap/jobs/cloud_controller_clock/bin/b-post-backup-unlock",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PostBackupUnlockOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-post-backup-unlock"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PostBackupUnlockOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all b-post-backup-unlock scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-post-backup-unlock",
				"/var/vcap/jobs/cloud_controller/bin/b-post-backup-unlock",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PostBackupUnlockOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/b-post-backup-unlock",
				"/var/vcap/jobs/cloud_controller/bin/b-post-backup-unlock",
			}))
		})
	})
})
