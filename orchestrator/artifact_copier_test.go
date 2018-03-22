package orchestrator_test

import (
	"fmt"
	executorFakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor/fakes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/executor"
)

var _ = Describe("ArtifactCopier", func() {
	var (
		artifactCopier orchestrator.ArtifactCopier
		logger         *fakes.FakeLogger
		deployment     *fakes.FakeDeployment
		localBackup    *fakes.FakeBackup
		fakeExecutor   *executorFakes.FakeExecutor
		err            error

		instance1 *fakes.FakeInstance
		instance2 *fakes.FakeInstance

		remoteBackup1 *fakes.FakeBackupArtifact
		remoteBackup2 *fakes.FakeBackupArtifact
	)

	BeforeEach(func() {
		logger = new(fakes.FakeLogger)
		fakeExecutor = new(executorFakes.FakeExecutor)

		instance1 = new(fakes.FakeInstance)
		instance2 = new(fakes.FakeInstance)

		deployment = new(fakes.FakeDeployment)

		localBackup = new(fakes.FakeBackup)

		remoteBackup1 = new(fakes.FakeBackupArtifact)
		remoteBackup2 = new(fakes.FakeBackupArtifact)

		artifactCopier = orchestrator.NewArtifactCopier(fakeExecutor, logger)
	})

	Context("DownloadBackupFromDeployment", func() {
		BeforeEach(func() {
			instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{remoteBackup1})
			instance2.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{remoteBackup2})
			deployment.BackupableInstancesReturns([]orchestrator.Instance{instance1, instance2})
		})

		JustBeforeEach(func() {
			err = artifactCopier.DownloadBackupFromDeployment(localBackup, deployment)
		})

		It("downloads the backup from deployment", func() {
			By("not failing", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			By("determining the backupable instances", func() {
				Expect(deployment.BackupableInstancesCallCount()).To(Equal(1))
			})

			By("checking the artifacts to backup in each instance", func() {
				Expect(instance1.ArtifactsToBackupCallCount()).To(Equal(1))
				Expect(instance2.ArtifactsToBackupCallCount()).To(Equal(1))
			})

			By("running the executor with the executables", func() {
				Expect(fakeExecutor.RunCallCount()).To(Equal(1))
				Expect(fakeExecutor.RunArgsForCall(0)).To(Equal([][]executor.Executable{{
					orchestrator.NewBackupDownloadExecutable(localBackup, remoteBackup1, logger),
					orchestrator.NewBackupDownloadExecutable(localBackup, remoteBackup2, logger),
				}}))
			})
		})

		Context("When the executor fails to run", func() {
			BeforeEach(func() {
				fakeExecutor.RunReturns([]error{fmt.Errorf("run error")})
			})

			It("should fail", func() {
				Expect(err).To(MatchError(ContainSubstring("run error")))
			})
		})
	})

	Context("UploadBackupToDeployment", func() {
		BeforeEach(func() {
			instance1.ArtifactsToRestoreReturns([]orchestrator.BackupArtifact{remoteBackup1})
			instance2.ArtifactsToRestoreReturns([]orchestrator.BackupArtifact{remoteBackup2})
			deployment.RestorableInstancesReturns([]orchestrator.Instance{instance1, instance2})
		})

		JustBeforeEach(func() {
			err = artifactCopier.UploadBackupToDeployment(localBackup, deployment)
		})

		It("uploads the backup to the deployment", func() {
			By("not failing", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			By("determining the restorable instances", func() {
				Expect(deployment.RestorableInstancesCallCount()).To(Equal(1))
			})

			By("checking the artifacts to restore for each instance", func() {
				Expect(instance1.ArtifactsToRestoreCallCount()).To(Equal(1))
				Expect(instance2.ArtifactsToRestoreCallCount()).To(Equal(1))
			})

			By("running the executor with the executables", func() {
				Expect(fakeExecutor.RunCallCount()).To(Equal(1))
				Expect(fakeExecutor.RunArgsForCall(0)).To(Equal([][]executor.Executable{{
					orchestrator.NewBackupUploadExecutable(localBackup, remoteBackup1, instance1, logger),
					orchestrator.NewBackupUploadExecutable(localBackup, remoteBackup2, instance2, logger),
				}}))
			})
		})

		Context("When the executor fails to run", func() {
			BeforeEach(func() {
				fakeExecutor.RunReturns([]error{fmt.Errorf("run error")})
			})
			It("should fail", func() {
				Expect(err).To(MatchError(ContainSubstring("run error")))
			})
		})
	})
})
