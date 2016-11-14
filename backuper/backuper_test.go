package backuper_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"
)

var _ = Describe("Backuper", func() {
	var (
		boshDirector           *fakes.FakeBoshDirector
		b                      *backuper.Backuper
		instance               *fakes.FakeInstance
		instances              backuper.Instances
		artifact               *fakes.FakeArtifact
		artifactCreator        *fakes.FakeArtifactCreator
		logger                 *fakes.FakeLogger
		deploymentName         = "foobarbaz"
		actualBackupError      error
		backupWriter           *fakes.FakeWriteCloser
		expectedLocalChecksum  = map[string]string{"file": "checksum"}
		expectedRemoteChecksum = map[string]string{"file": "checksum"}
	)

	BeforeEach(func() {
		boshDirector = new(fakes.FakeBoshDirector)
		artifactCreator = new(fakes.FakeArtifactCreator)
		artifact = new(fakes.FakeArtifact)
		logger = new(fakes.FakeLogger)
		instance = new(fakes.FakeInstance)
		instances = backuper.Instances{instance}
		backupWriter = new(fakes.FakeWriteCloser)
		b = backuper.New(boshDirector, artifactCreator.Spy, logger)
	})
	JustBeforeEach(func() {
		actualBackupError = b.Backup(deploymentName)
	})

	Context("backups up an instance", func() {
		BeforeEach(func() {
			artifactCreator.Returns(artifact, nil)
			boshDirector.FindInstancesReturns(instances, nil)
			instance.IsBackupableReturns(true, nil)
			instance.CleanupReturns(nil)
			instance.NameReturns("redis")
			instance.IDReturns("0")
			artifact.CreateFileReturns(backupWriter, nil)
			instance.StreamBackupFromRemoteReturns(nil)
			artifact.CalculateChecksumReturns(expectedLocalChecksum, nil)
			instance.BackupChecksumReturns(expectedRemoteChecksum, nil)
		})

		It("does not fail", func() {
			Expect(actualBackupError).ToNot(HaveOccurred())
		})

		It("finds a instances for the deployment", func() {
			Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
			Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the instance is backupable", func() {
			Expect(instance.IsBackupableCallCount()).To(Equal(1))
		})

		It("runs backup scripts on the instance", func() {
			Expect(instance.BackupCallCount()).To(Equal(1))
		})

		It("ensures that instance is cleaned up", func() {
			Expect(instance.CleanupCallCount()).To(Equal(1))
		})

		It("creates a local artifact", func() {
			Expect(artifactCreator.CallCount()).To(Equal(1))
		})

		It("names the artifact after the deployment", func() {
			Expect(artifactCreator.ArgsForCall(0)).To(Equal(deploymentName))
		})

		It("creates files on disk for each backupable instance", func() {
			Expect(artifact.CreateFileCallCount()).To(Equal(1))
			Expect(artifact.CreateFileArgsForCall(0)).To(Equal(instance))
		})

		It("streams the contents to the writer", func() {
			Expect(instance.StreamBackupFromRemoteCallCount()).To(Equal(1))
			Expect(instance.StreamBackupFromRemoteArgsForCall(0)).To(Equal(backupWriter))
		})
		It("adds the checksum for the instance to the metadata", func() {
			Expect(artifact.AddChecksumCallCount()).To(Equal(1))
			actualInstance, actualShasum := artifact.AddChecksumArgsForCall(0)
			Expect(actualInstance).To(Equal(instance))
			Expect(actualShasum).To(Equal(expectedLocalChecksum))
		})

		It("validates remote and local checksums match", func() {
			Expect(instance.BackupChecksumCallCount()).To(Equal(1))
		})
	})

	Context("backups deployment with a non backupable instance and a backupable instance", func() {
		var nonBackupableInstance *fakes.FakeInstance

		BeforeEach(func() {
			nonBackupableInstance = new(fakes.FakeInstance)
			instances = backuper.Instances{instance, nonBackupableInstance}

			artifactCreator.Returns(artifact, nil)
			artifact.CreateFileReturns(backupWriter, nil)
			boshDirector.FindInstancesReturns(instances, nil)
			instance.IsBackupableReturns(true, nil)
			instance.CleanupReturns(nil)
			instance.NameReturns("redis")
			instance.IDReturns("0")

			nonBackupableInstance.IsBackupableReturns(false, nil)
			nonBackupableInstance.NameReturns("broker")
			nonBackupableInstance.IDReturns("0")
		})

		It("does not fail", func() {
			Expect(actualBackupError).ToNot(HaveOccurred())
		})

		It("finds the instances for the deployment", func() {
			Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
			Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the instances is backupable", func() {
			Expect(instance.IsBackupableCallCount()).To(Equal(1))
			Expect(nonBackupableInstance.IsBackupableCallCount()).To(Equal(1))
		})

		It("runs backup scripts on the backupable instance", func() {
			Expect(instance.BackupCallCount()).To(Equal(1))
		})

		It("does not run the backup scripts on the non backupable instance", func() {
			Expect(nonBackupableInstance.BackupCallCount()).To(BeZero())
		})

		It("ensures that instance is cleaned up", func() {
			Expect(instance.CleanupCallCount()).To(Equal(1))
		})

		It("ensures that nonbackupable instance is cleaned up", func() {
			Expect(nonBackupableInstance.CleanupCallCount()).To(Equal(1))
		})

		It("creates files on disk for only the backupable instance", func() {
			Expect(artifact.CreateFileCallCount()).To(Equal(1))
			Expect(artifact.CreateFileArgsForCall(0)).To(Equal(instance))
		})
	})

	Describe("failures", func() {
		var expectedError = fmt.Errorf("Jesus!")
		Context("fails to find instances", func() {
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(nil, expectedError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(expectedError))
			})
		})

		Context("fails when checking if instances are backupable", func() {
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(false, expectedError)
			})

			It("finds instances with the deployment name", func() {
				Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
				Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(expectedError))
			})
			It("ensures that deployment is cleaned up", func() {
				Expect(instance.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("fails if the deployment is not backupable", func() {
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(false, nil)
			})

			It("finds a instances with the deployment name", func() {
				Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
				Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
			})
			It("checks if the deployment is backupable", func() {
				Expect(instance.IsBackupableCallCount()).To(Equal(1))
			})
			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError("Deployment '" + deploymentName + "' has no backup scripts"))
			})
			It("ensures that deployment is cleaned up", func() {
				Expect(instance.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("fails if backup cannot be drained", func() {
			var drainError = fmt.Errorf("they are bringing crime")
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(true, nil)
				artifactCreator.Returns(artifact, nil)
				instance.StreamBackupFromRemoteReturns(drainError)
			})

			It("check if the deployment is backupable", func() {
				Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
				Expect(instance.IsBackupableCallCount()).To(Equal(1))
			})

			It("backs up the instance", func() {
				Expect(instance.BackupCallCount()).To(Equal(1))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(drainError))
			})
		})

		Context("fails if artifact cannot be created", func() {
			var artifactError = fmt.Errorf("they are bringing crime")
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(true, nil)

				artifactCreator.Returns(nil, artifactError)
			})

			It("check if the deployment is backupable", func() {
				Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
				Expect(instance.IsBackupableCallCount()).To(Equal(1))
			})

			It("dosent backup the instance", func() {
				Expect(instance.BackupCallCount()).To(BeZero())
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(artifactError))
			})
		})

		Context("fails if file cannot be created", func() {
			var fileError = fmt.Errorf("i have a very good brain")
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(true, nil)

				artifactCreator.Returns(artifact, nil)
				artifact.CreateFileReturns(nil, fileError)
			})

			It("check if the deployment is backupable", func() {
				Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
				Expect(instance.IsBackupableCallCount()).To(Equal(1))
			})

			It("does try to backup the instance", func() {
				Expect(instance.BackupCallCount()).To(Equal(1))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(fileError))
			})
		})

		Context("fails if backup is not a success", func() {
			var backupError = fmt.Errorf("i have the best words")
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(true, nil)

				artifactCreator.Returns(artifact, nil)
				instance.BackupReturns(backupError)
			})

			It("check if the deployment is backupable", func() {
				Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
				Expect(instance.IsBackupableCallCount()).To(Equal(1))
			})

			It("does try to backup the instance", func() {
				Expect(instance.BackupCallCount()).To(Equal(1))
			})

			It("does not try to create files in the artifact", func() {
				Expect(artifact.CreateFileCallCount()).To(BeZero())
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(backupError))
			})
		})

		Context("fails if local shasum calculation fails", func() {
			shasumError := fmt.Errorf("yuuuge")
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(true, nil)
				artifactCreator.Returns(artifact, nil)
				instance.BackupReturns(nil)
				artifact.CreateFileReturns(backupWriter, nil)

				artifact.CalculateChecksumReturns(nil, shasumError)
			})

			It("does try to create files in the artifact", func() {
				Expect(artifact.CreateFileCallCount()).To(Equal(1))
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(shasumError))
			})
		})

		Context("fails if the remote shasum cant be calulated", func() {
			remoteShasumError := fmt.Errorf("i have created so many jobs")
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(true, nil)
				artifactCreator.Returns(artifact, nil)
				instance.BackupReturns(nil)
				artifact.CreateFileReturns(backupWriter, nil)

				instance.BackupChecksumReturns(nil, remoteShasumError)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(remoteShasumError))
			})

			It("dosen't try to append shasum to metadata", func() {
				Expect(artifact.AddChecksumCallCount()).To(BeZero())
			})
		})
		Context("fails if the remote shasum dosen't match the local shasum", func() {
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(true, nil)
				artifactCreator.Returns(artifact, nil)
				instance.BackupReturns(nil)
				artifact.CreateFileReturns(backupWriter, nil)

				artifact.CalculateChecksumReturns(map[string]string{"file": "this won't match"}, nil)
				instance.BackupChecksumReturns(map[string]string{"file": "this wont match"}, nil)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(ContainSubstring("Backup artifact is corrupted")))
			})

			It("dosen't try to append shasum to metadata", func() {
				Expect(artifact.AddChecksumCallCount()).To(BeZero())
			})
		})

		Context("fails if the number of files in the artifact dont match", func() {
			BeforeEach(func() {
				boshDirector.FindInstancesReturns(instances, nil)
				instance.IsBackupableReturns(true, nil)
				artifactCreator.Returns(artifact, nil)
				instance.BackupReturns(nil)
				artifact.CreateFileReturns(backupWriter, nil)

				artifact.CalculateChecksumReturns(map[string]string{"file": "this will match", "extra": "this won't match"}, nil)
				instance.BackupChecksumReturns(map[string]string{"file": "this will match"}, nil)
			})

			It("fails the backup process", func() {
				Expect(actualBackupError).To(MatchError(ContainSubstring("Backup artifact is corrupted")))
			})

			It("dosen't try to append shasum to metadata", func() {
				Expect(artifact.AddChecksumCallCount()).To(BeZero())
			})
		})
	})
})

var _ = Describe("restore", func() {
	Context("restores an instance from backup", func() {
		var (
			restoreError    error
			artifactCreator *fakes.FakeArtifactCreator
			artifact        *fakes.FakeArtifact
			boshDirector    *fakes.FakeBoshDirector
			logger          *fakes.FakeLogger
			instance        *fakes.FakeInstance
			instances       backuper.Instances
			b               *backuper.Backuper
			deploymentName  string
		)

		BeforeEach(func() {
			instance = new(fakes.FakeInstance)
			instances = backuper.Instances{instance}
			boshDirector = new(fakes.FakeBoshDirector)
			logger = new(fakes.FakeLogger)
			artifactCreator = new(fakes.FakeArtifactCreator)
			artifact = new(fakes.FakeArtifact)

			artifactCreator.Returns(artifact, nil)
			boshDirector.FindInstancesReturns(instances, nil)
			instance.IsRestorableReturns(true, nil)
			artifact.DeploymentMatchesReturns(true, nil)

			b = backuper.New(boshDirector, artifactCreator.Spy, logger)

			deploymentName = "deployment-to-restore"
		})

		JustBeforeEach(func() {
			restoreError = b.Restore(deploymentName)
		})

		It("does not fail", func() {
			Expect(restoreError).NotTo(HaveOccurred())
		})

		It("ensures that instance is cleaned up", func() {
			Expect(instance.CleanupCallCount()).To(Equal(1))
		})

		It("finds a instances for the deployment", func() {
			Expect(boshDirector.FindInstancesCallCount()).To(Equal(1))
			Expect(boshDirector.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the instance is restorable", func() {
			Expect(instance.IsRestorableCallCount()).To(Equal(1))
		})

		It("checks that the deployment topology matches the topology of the backup", func() {
			Expect(artifactCreator.CallCount()).To(Equal(1))
			Expect(artifact.DeploymentMatchesCallCount()).To(Equal(1))

			name, instances := artifact.DeploymentMatchesArgsForCall(0)
			Expect(name).To(Equal(deploymentName))
			Expect(instances).To(ContainElement(instance))
		})

		XIt("streams the local backup to the instance")

		It("calls restore on the instance", func() {
			Expect(instance.RestoreCallCount()).To(Equal(1))
		})

		Describe("failures", func() {
			Context("fails to find instances", func() {
				BeforeEach(func() {
					boshDirector.FindInstancesReturns(nil, fmt.Errorf("they will pay for the wall"))
				})

				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(MatchError("they will pay for the wall"))
				})
			})

			Context("if no instances are restorable", func() {
				BeforeEach(func() {
					instance.IsRestorableReturns(false, nil)
				})

				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(HaveOccurred())
				})
			})

			Context("if checking the instance's restorable status fails", func() {
				BeforeEach(func() {
					instance.IsRestorableReturns(true, fmt.Errorf("the beauty of me is that I'm very rich"))
				})
				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(HaveOccurred())
				})
			})

			Context("if the deployment's topology doesn't match that of the backup", func() {
				BeforeEach(func() {
					artifact.DeploymentMatchesReturns(false, nil)
				})

				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(HaveOccurred())
				})
			})

			Context("if checking the deployment topology fails", func() {
				BeforeEach(func() {
					artifact.DeploymentMatchesReturns(true, fmt.Errorf("my fingers are long and beautiful"))
				})

				It("returns an error", func() {
					actualError := b.Restore(deploymentName)
					Expect(actualError).To(HaveOccurred())
				})
			})
		})
	})
})
