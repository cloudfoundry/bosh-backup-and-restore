package backuper_test

import (
	"bytes"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper/fakes"
)

var _ = Describe("Backuper", func() {
	var (
		boshDirector      *fakes.FakeBoshDirector
		b                 *backuper.Backuper
		instance          *fakes.FakeInstance
		instances         backuper.Instances
		artifact          *fakes.FakeArtifact
		artifactCreator   *fakes.FakeArtifactCreator
		deploymentName    = "foobarbaz"
		actualBackupError error
	)

	BeforeEach(func() {
		boshDirector = new(fakes.FakeBoshDirector)
		artifactCreator = new(fakes.FakeArtifactCreator)
		artifact = new(fakes.FakeArtifact)
		instance = new(fakes.FakeInstance)
		instances = backuper.Instances{instance}
		b = backuper.New(boshDirector, artifactCreator.Spy)
	})
	JustBeforeEach(func() {
		actualBackupError = b.Backup(deploymentName)
	})

	Context("backups up an instance", func() {
		var expectedReader io.Reader
		BeforeEach(func() {
			expectedReader = bytes.NewBufferString("some data")

			artifactCreator.Returns(artifact, nil)
			boshDirector.FindInstancesReturns(instances, nil)
			instance.IsBackupableReturns(true, nil)
			instance.CleanupReturns(nil)
			instance.NameReturns("redis")
			instance.IDReturns("0")
			instance.DrainBackupReturns(expectedReader, nil)
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
			filename, _ := artifact.CreateFileArgsForCall(0)
			Expect(filename).To(Equal("redis-0.tgz"))
		})
		It("writes the drained backup to file", func() {
			Expect(instance.DrainBackupCallCount()).To(Equal(1))
			Expect(artifact.CreateFileCallCount()).To(Equal(1))
			filename, reader := artifact.CreateFileArgsForCall(0)
			Expect(filename).To(Equal("redis-0.tgz"))
			Expect(reader).To(Equal(expectedReader))
		})
	})

	Context("backups deployment with a non backupable instance and a backupable instance", func() {
		var nonBackupableInstance *fakes.FakeInstance

		BeforeEach(func() {
			nonBackupableInstance = new(fakes.FakeInstance)
			instances = backuper.Instances{instance, nonBackupableInstance}

			artifactCreator.Returns(artifact, nil)
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
			Expect(artifact.CreateFileArgsForCall(0)).To(Equal("redis-0.tgz"))
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
				instance.DrainBackupReturns(nil, drainError)
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

			It("does not try to create files in the artifact", func() {
				Expect(artifact.CreateFileCallCount()).To(BeZero())
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
				artifact.CreateFileReturns(fileError)
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
	})
})
