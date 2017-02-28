package instance_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
)

var _ = Describe("Jobs", func() {
	var jobs instance.Jobs
	var scripts instance.BackupAndRestoreScripts
	var artifactNames map[string]instance.Metadata

	BeforeEach(func() {
		artifactNames = map[string]instance.Metadata{}
	})

	JustBeforeEach(func() {
		jobs = instance.NewJobs(scripts, artifactNames)
	})

	Describe("NewJobs", func() {
		Context("when there are two jobs each with a backup script", func() {
			BeforeEach(func() {
				scripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/b-backup",
					"/var/vcap/jobs/bar/bin/b-backup",
				}
			})
			It("groups scripts to create jobs", func() {
				Expect(jobs).To(ConsistOf(
					instance.NewJob(instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/b-backup"}, instance.Metadata{}),
					instance.NewJob(instance.BackupAndRestoreScripts{"/var/vcap/jobs/bar/bin/b-backup"}, instance.Metadata{}),
				))
			})
		})

		Context("when there is one job with a backup script", func() {
			BeforeEach(func() {
				scripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/b-backup",
				}
			})
			It("groups scripts to create jobs", func() {
				Expect(jobs).To(ConsistOf(
					instance.NewJob(instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/b-backup"}, instance.Metadata{}),
				))
			})
		})

		Context("when there is one job with a backup script and an blob name", func() {
			BeforeEach(func() {
				scripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/b-backup",
				}
				artifactNames = map[string]instance.Metadata{
					"foo": {
						BackupName: "a-bosh-backup",
					},
				}
			})

			It("creates a job with the correct blob name", func() {
				Expect(jobs).To(ConsistOf(
					instance.NewJob(
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/b-backup"},
						instance.Metadata{
							BackupName: "a-bosh-backup",
						},
					),
				))
			})
		})

		Context("when there are two jobs, both with backup scripts and unique metadata names", func() {
			BeforeEach(func() {
				scripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/b-backup",
					"/var/vcap/jobs/bar/bin/b-backup",
				}
				artifactNames = map[string]instance.Metadata{
					"foo": {
						BackupName: "a-bosh-backup",
					},
					"bar": {
						BackupName: "another-backup",
					},
				}
			})

			It("creates two jobs with the correct blob names", func() {
				Expect(jobs).To(ConsistOf(
					instance.NewJob(
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/b-backup"},
						instance.Metadata{
							BackupName: "a-bosh-backup",
						},
					),
					instance.NewJob(
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/bar/bin/b-backup"},
						instance.Metadata{
							BackupName: "another-backup",
						},
					),
				))
			})
		})

	})

	Context("contains jobs with backup script", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/b-backup",
				"/var/vcap/jobs/bar/bin/b-restore",
			}
		})

		Describe("Backupable", func() {
			It("returns the backupable job", func() {
				Expect(jobs.Backupable()).To(ConsistOf(
					instance.NewJob(instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/b-backup"}, instance.Metadata{}),
				))
			})
		})

		Describe("AnyAreBackupable", func() {
			It("returns true", func() {
				Expect(jobs.AnyAreBackupable()).To(BeTrue())
			})
		})
	})

	Context("contains no jobs with backup script", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/b-restore",
			}
		})

		Describe("Backupable", func() {
			It("returns empty", func() {
				Expect(jobs.Backupable()).To(BeEmpty())
			})
		})

		Describe("AnyAreBackupable", func() {
			It("returns false", func() {
				Expect(jobs.AnyAreBackupable()).To(BeFalse())
			})
		})
	})

	Context("contains jobs with pre-backup-lock scripts", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/b-pre-backup-lock",
				"/var/vcap/jobs/foo/bin/b-backup",
				"/var/vcap/jobs/bar/bin/b-restore",
			}
		})

		Describe("PreBackupable", func() {
			It("returns the lockable job", func() {
				Expect(jobs.PreBackupable()).To(ConsistOf(instance.NewJob(
					instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/b-pre-backup-lock",
						"/var/vcap/jobs/foo/bin/b-backup",
					}, instance.Metadata{}),
				))
			})
		})

		Describe("AnyArePreBackupable", func() {
			It("returns true", func() {
				Expect(jobs.AnyArePreBackupable()).To(BeTrue())
			})
		})
	})
	Context("contains no jobs with pre-backup-lock scripts", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/b-restore",
			}
		})
		Describe("PreBackupable", func() {
			It("returns empty", func() {
				Expect(jobs.PreBackupable()).To(BeEmpty())
			})
		})

		Describe("AnyArePreBackupable", func() {
			It("returns false", func() {
				Expect(jobs.AnyArePreBackupable()).To(BeFalse())
			})
		})
	})

	Context("contains jobs with post-backup-lock scripts", func() {

		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/b-backup",
				"/var/vcap/jobs/foo/bin/b-post-backup-unlock",
				"/var/vcap/jobs/bar/bin/b-restore",
			}
		})

		Describe("PostBackupable", func() {
			It("returns the unlockable job", func() {
				Expect(jobs.PostBackupable()).To(ConsistOf(instance.NewJob(
					instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/b-post-backup-unlock",
						"/var/vcap/jobs/foo/bin/b-backup",
					}, instance.Metadata{}),
				))
			})
		})

		Describe("AnyArePostBackupable", func() {
			It("returns true", func() {
				Expect(jobs.AnyArePostBackupable()).To(BeTrue())
			})
		})
	})
	Context("contains no jobs with backup script", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/b-restore",
			}
		})

		Describe("PostBackupable", func() {
			It("returns empty", func() {
				Expect(jobs.PostBackupable()).To(BeEmpty())
			})
		})
	})

	Context("contains jobs with restore scripts", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/b-backup",
				"/var/vcap/jobs/foo/bin/b-post-backup-unlock",
				"/var/vcap/jobs/bar/bin/b-restore",
			}
		})

		Describe("Restorable", func() {
			It("returns the unlockable job", func() {
				Expect(jobs.Restorable()).To(ConsistOf(instance.NewJob(
					instance.BackupAndRestoreScripts{"/var/vcap/jobs/bar/bin/b-restore"}, instance.Metadata{}),
				))
			})
		})

		Describe("AnyAreRestorable", func() {
			It("returns true", func() {
				Expect(jobs.AnyAreRestorable()).To(BeTrue())
			})
		})

		Describe("AnyNeedDefaultBlobsForRestore", func() {
			It("returns true, as all of the jobs need a default blob for restore", func() {
				Expect(jobs.AnyNeedDefaultBlobsForRestore()).To(BeTrue())
			})
		})
	})

	Context("contains no jobs with backup script", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/b-backup",
			}
		})

		It("returns empty", func() {
			Expect(jobs.Restorable()).To(BeEmpty())
		})
	})

	Context("contains no jobs with named backup blobs", func() {
		Describe("WithNamedBackupBlobs", func() {
			It("returns empty", func() {
				Expect(jobs.WithNamedBackupBlobs()).To(BeEmpty())
			})
		})

		Describe("BackupBlobNames", func() {
			It("returns empty", func() {
				Expect(jobs.BackupBlobNames()).To(BeEmpty())
			})
		})
	})

	Context("contains jobs with a named backup blob", func() {
		BeforeEach(func() {
			artifactNames = map[string]instance.Metadata{
				"bar": {
					BackupName: "my-cool-blob",
				},
			}
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/b-backup",
				"/var/vcap/jobs/bar/bin/b-restore",
				"/var/vcap/jobs/foo/bin/b-backup",
				"/var/vcap/jobs/baz/bin/b-restore",
			}
		})

		Describe("AnyNeedDefaultBlobsForBackup", func() {
			It("returns true", func() {
				Expect(jobs.AnyNeedDefaultBlobsForBackup()).To(BeTrue())
			})
		})

		Describe("WithNamedBackupBlobs", func() {
			It("returns jobs with named backup blobs", func() {
				Expect(jobs.WithNamedBackupBlobs()).To(ConsistOf(instance.NewJob(
					instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/b-backup",
						"/var/vcap/jobs/bar/bin/b-restore",
					}, instance.Metadata{
						BackupName: "my-cool-blob",
					}),
				))
			})
		})
	})

	Context("contains jobs with a named restore blob", func() {
		BeforeEach(func() {
			artifactNames = map[string]instance.Metadata{
				"bar": {
					RestoreName: "my-cool-restore",
				},
			}
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/b-backup",
				"/var/vcap/jobs/bar/bin/b-restore",
				"/var/vcap/jobs/foo/bin/b-backup",
				"/var/vcap/jobs/baz/bin/b-restore",
			}
		})

		Describe("NamedRestoreBlobs", func() {
			It("returns a list of blob names", func() {
				Expect(jobs.NamedRestoreBlobs()).To(ConsistOf("my-cool-restore"))
			})
		})

		Describe("WithNamedRestoreBlobs", func() {
			It("returns jobs with named backup blobs", func() {
				Expect(jobs.WithNamedRestoreBlobs()).To(ConsistOf(instance.NewJob(
					instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/b-backup",
						"/var/vcap/jobs/bar/bin/b-restore",
					}, instance.Metadata{
						RestoreName: "my-cool-restore",
					}),
				))
			})
		})

		Describe("AnyNeedDefaultBlobsForRestore", func() {
			It("returns true, as job 'baz' needs a default blob for restore", func() {
				Expect(jobs.AnyNeedDefaultBlobsForRestore()).To(BeTrue())
			})
		})
	})

	Context("contains jobs with multiple named blobs", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/b-backup",
				"/var/vcap/jobs/bar/bin/b-backup",
			}
			artifactNames = map[string]instance.Metadata{
				"foo": {
					BackupName: "a-bosh-backup",
				},
				"bar": {
					BackupName: "another-backup",
				},
			}
		})

		Describe("BackupBlobNames", func() {
			It("returns a list of blob names", func() {
				Expect(jobs.BackupBlobNames()).To(ConsistOf("a-bosh-backup", "another-backup"))
			})
		})

		Describe("AnyNeedDefaultBlobsForRestore", func() {
			It("returns false, as none of the jobs need a default blob for restore", func() {
				Expect(jobs.AnyNeedDefaultBlobsForRestore()).To(BeFalse())
			})
		})

		Describe("AnyNeedDefaultBlobsForBackup", func() {
			It("returns false", func() {
				Expect(jobs.AnyNeedDefaultBlobsForBackup()).To(BeFalse())
			})
		})
	})
})
