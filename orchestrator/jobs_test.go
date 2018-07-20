package orchestrator_test

import (
	"log"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	orchestratorFakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Jobs", func() {
	var jobs orchestrator.Jobs
	var logger boshlog.Logger

	BeforeEach(func() {
		combinedLog := log.New(GinkgoWriter, "[instance-test] ", log.Lshortfile)
		logger = boshlog.New(boshlog.LevelDebug, combinedLog)
	})

	Context("contains jobs with backup script", func() {
		var backupableJob *orchestratorFakes.FakeJob
		var nonBackupableJob *orchestratorFakes.FakeJob

		BeforeEach(func() {
			backupableJob = new(orchestratorFakes.FakeJob)
			backupableJob.HasBackupReturns(true)

			nonBackupableJob = new(orchestratorFakes.FakeJob)
			nonBackupableJob.HasBackupReturns(false)

			jobs = orchestrator.Jobs([]orchestrator.Job{
				backupableJob,
				nonBackupableJob,
			})
		})

		Describe("Backupable", func() {
			It("returns the backupable job", func() {
				Expect(jobs.Backupable()).To(ConsistOf(backupableJob))
			})
		})

		Describe("AnyAreBackupable", func() {
			It("returns true", func() {
				Expect(jobs.AnyAreBackupable()).To(BeTrue())
			})
		})
	})

	Context("contains no jobs with backup script", func() {
		var nonBackupableJob *orchestratorFakes.FakeJob

		BeforeEach(func() {
			nonBackupableJob = new(orchestratorFakes.FakeJob)
			nonBackupableJob.HasBackupReturns(false)

			jobs = orchestrator.Jobs([]orchestrator.Job{
				nonBackupableJob,
			})
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

	Context("contains jobs with restore scripts", func() {
		var restorableJob *orchestratorFakes.FakeJob
		var nonRestorableJob *orchestratorFakes.FakeJob

		BeforeEach(func() {
			restorableJob = new(orchestratorFakes.FakeJob)
			restorableJob.HasRestoreReturns(true)

			nonRestorableJob = new(orchestratorFakes.FakeJob)
			nonRestorableJob.HasRestoreReturns(false)

			jobs = orchestrator.Jobs([]orchestrator.Job{
				restorableJob,
				nonRestorableJob,
			})
		})

		Describe("Restorable", func() {
			It("returns the restorable job", func() {
				Expect(jobs.Restorable()).To(ConsistOf(restorableJob))
			})
		})

		Describe("AnyAreRestorable", func() {
			It("returns true", func() {
				Expect(jobs.AnyAreRestorable()).To(BeTrue())
			})
		})
	})

	Context("contains no jobs with restore script", func() {
		var nonRestorableJob *orchestratorFakes.FakeJob

		BeforeEach(func() {
			nonRestorableJob = new(orchestratorFakes.FakeJob)
			nonRestorableJob.HasRestoreReturns(false)

			jobs = orchestrator.Jobs([]orchestrator.Job{
				nonRestorableJob,
			})
		})

		Describe("Restorable", func() {
			It("returns empty", func() {
				Expect(jobs.Restorable()).To(BeEmpty())
			})
		})

		Describe("AnyAreRestorable", func() {
			It("returns false", func() {
				Expect(jobs.AnyAreRestorable()).To(BeFalse())
			})
		})
	})

	Context("contains no jobs with named backup artifacts", func() {
		Describe("CustomBackupArtifactNames", func() {
			It("returns empty", func() {
				Expect(jobs.CustomBackupArtifactNames()).To(BeEmpty())
			})
		})
	})

	Context("contains jobs with a named backup artifact", func() {
		var jobWithNamedBackupArtifact, anotherJobWithNamedBackupArtifact *orchestratorFakes.FakeJob
		var jobWithoutNamedBackupArtifact *orchestratorFakes.FakeJob

		BeforeEach(func() {
			jobWithNamedBackupArtifact = new(orchestratorFakes.FakeJob)
			jobWithNamedBackupArtifact.HasNamedBackupArtifactReturns(true)
			jobWithNamedBackupArtifact.BackupArtifactNameReturns("backup-artifact-name")

			anotherJobWithNamedBackupArtifact = new(orchestratorFakes.FakeJob)
			anotherJobWithNamedBackupArtifact.HasNamedBackupArtifactReturns(true)
			anotherJobWithNamedBackupArtifact.BackupArtifactNameReturns("another-backup-artifact-name")

			jobWithoutNamedBackupArtifact = new(orchestratorFakes.FakeJob)
			jobWithoutNamedBackupArtifact.HasNamedBackupArtifactReturns(false)

			jobs = orchestrator.Jobs([]orchestrator.Job{
				jobWithNamedBackupArtifact,
				anotherJobWithNamedBackupArtifact,
				jobWithoutNamedBackupArtifact,
			})
		})

		Describe("CustomBackupArtifactNames", func() {
			It("returns a list of artifact names", func() {
				Expect(jobs.CustomBackupArtifactNames()).To(ConsistOf(
					"backup-artifact-name",
					"another-backup-artifact-name",
				))
			})
		})
	})

	Context("contains jobs with a named restore artifact", func() {
		var jobWithNamedRestoreArtifact *orchestratorFakes.FakeJob
		var jobWithoutNamedRestoreArtifact *orchestratorFakes.FakeJob

		BeforeEach(func() {
			jobWithNamedRestoreArtifact = new(orchestratorFakes.FakeJob)
			jobWithNamedRestoreArtifact.HasNamedRestoreArtifactReturns(true)
			jobWithNamedRestoreArtifact.RestoreArtifactNameReturns("restore-artifact-name")

			jobWithoutNamedRestoreArtifact = new(orchestratorFakes.FakeJob)
			jobWithoutNamedRestoreArtifact.HasNamedRestoreArtifactReturns(false)

			jobs = orchestrator.Jobs([]orchestrator.Job{
				jobWithNamedRestoreArtifact,
				jobWithoutNamedRestoreArtifact,
			})
		})

		Describe("CustomRestoreArtifactNames", func() {
			It("returns a list of artifact names", func() {
				Expect(jobs.CustomRestoreArtifactNames()).To(ConsistOf("restore-artifact-name"))
			})
		})
	})

})
