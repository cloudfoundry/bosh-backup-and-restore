package standalone_test

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/standalone"

	"io/ioutil"

	instancefakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance/fakes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	sshfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeploymentManager", func() {
	var deploymentManager DeploymentManager
	var deploymentName = "bosh"
	var artifact *fakes.FakeBackup
	var logger *fakes.FakeLogger
	var hostName = "hostname"
	var username = "username"
	var privateKey string
	var fakeJobFinder *instancefakes.FakeJobFinder
	var remoteRunnerFactory *sshfakes.FakeRemoteRunnerFactory
	var remoteRunner *sshfakes.FakeRemoteRunner

	BeforeEach(func() {
		privateKey = createTempFile("privateKey")
		logger = new(fakes.FakeLogger)
		artifact = new(fakes.FakeBackup)
		remoteRunnerFactory = new(sshfakes.FakeRemoteRunnerFactory)
		fakeJobFinder = new(instancefakes.FakeJobFinder)
		remoteRunner = new(sshfakes.FakeRemoteRunner)

		deploymentManager = NewDeploymentManager(logger, hostName, username, privateKey, fakeJobFinder, remoteRunnerFactory.Spy)
	})

	AfterEach(func() {
		os.Remove(privateKey)
	})

	Describe("Find", func() {
		var actualDeployment orchestrator.Deployment
		var actualError error
		var fakeJobs orchestrator.Jobs

		JustBeforeEach(func() {
			actualDeployment, actualError = deploymentManager.Find(deploymentName)
		})

		Context("success", func() {
			BeforeEach(func() {
				fakeJobs = orchestrator.Jobs{instance.NewJob(nil, "", nil, "", instance.BackupAndRestoreScripts{"foo"}, instance.Metadata{})}
				remoteRunnerFactory.Returns(remoteRunner, nil)
				fakeJobFinder.FindJobsReturns(fakeJobs, nil)
			})
			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("invokes connection creator", func() {
				Expect(remoteRunnerFactory.CallCount()).To(Equal(1))
			})

			It("invokes job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(1))
			})

			It("returns a deployment", func() {
				Expect(actualDeployment).To(Equal(orchestrator.NewDeployment(logger, []orchestrator.Instance{
					NewDeployedInstance("bosh", remoteRunner, logger, fakeJobs, false),
				})))
			})
		})

		Context("can't read private key", func() {
			BeforeEach(func() {
				os.Remove(privateKey)
			})

			It("should fail", func() {
				Expect(actualError).To(MatchError(ContainSubstring("failed reading private key")))
			})

			It("should not invoke connection creator", func() {
				Expect(remoteRunnerFactory.CallCount()).To(BeZero())
			})
		})

		Context("can't create SSH connection", func() {
			connError := fmt.Errorf("error")

			BeforeEach(func() {
				remoteRunnerFactory.Returns(nil, connError)
			})

			It("should fail", func() {
				Expect(actualError).To(MatchError(connError))
			})

			It("should invoke connection creator", func() {
				Expect(remoteRunnerFactory.CallCount()).To(Equal(1))
			})

			It("should not invoke job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(BeZero())
			})

		})

		Context("can't find jobs", func() {
			findJobsErr := fmt.Errorf("error")

			BeforeEach(func() {
				remoteRunnerFactory.Returns(remoteRunner, nil)
				fakeJobFinder.FindJobsReturns(nil, findJobsErr)
			})

			It("should fail", func() {
				Expect(actualError).To(MatchError(findJobsErr))
			})

			It("should invoke connection creator", func() {
				Expect(remoteRunnerFactory.CallCount()).To(Equal(1))
			})

			It("should not invoke job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(1))
			})
		})

	})

	Describe("SaveManifest", func() {
		It("does nothing", func() {
			err := deploymentManager.SaveManifest(deploymentName, artifact)
			Expect(err).NotTo(HaveOccurred())
		})
	})

})

var _ = Describe("DeployedInstance", func() {
	var logger *fakes.FakeLogger
	var remoteRunner *sshfakes.FakeRemoteRunner
	var inst DeployedInstance
	var artifactDirCreated bool

	BeforeEach(func() {
		logger = new(fakes.FakeLogger)
		remoteRunner = new(sshfakes.FakeRemoteRunner)
	})

	Describe("Cleanup", func() {
		var err error

		JustBeforeEach(func() {
			inst = NewDeployedInstance("group", remoteRunner, logger, []orchestrator.Job{}, artifactDirCreated)
			err = inst.Cleanup()
		})

		BeforeEach(func() {
			artifactDirCreated = true
		})

		It("does not fail", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes the artifact directory", func() {
			Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(1))
			Expect(remoteRunner.RemoveDirectoryArgsForCall(0)).To(Equal("/var/vcap/store/bbr-backup"))
		})

		Context("when the artifact directory was not created this time", func() {
			BeforeEach(func() {
				artifactDirCreated = false
			})

			It("does not remove the artifact directory", func() {
				Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(0))
			})
		})

		Context("when cleanup fails", func() {
			BeforeEach(func() {
				remoteRunner.RemoveDirectoryReturns(fmt.Errorf("fool!"))
			})

			It("returns an error", func() {
				Expect(err).To(SatisfyAll(
					MatchError(ContainSubstring("Unable to clean up backup artifact")),
					MatchError(ContainSubstring("fool!")),
				))
			})
		})
	})

	Describe("CleanupPrevious", func() {
		var err error

		JustBeforeEach(func() {
			inst = NewDeployedInstance("group", remoteRunner, logger, []orchestrator.Job{}, artifactDirCreated)
			err = inst.CleanupPrevious()
		})

		BeforeEach(func() {
			artifactDirCreated = true
		})

		It("does not fail", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes the artifact directory", func() {
			Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(1))
			Expect(remoteRunner.RemoveDirectoryArgsForCall(0)).To(Equal("/var/vcap/store/bbr-backup"))
		})

		Context("when the artifact directory was not created this time", func() {
			BeforeEach(func() {
				artifactDirCreated = false
			})

			It("does remove the artifact directory", func() {
				Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(1))
				Expect(remoteRunner.RemoveDirectoryArgsForCall(0)).To(Equal("/var/vcap/store/bbr-backup"))
			})
		})

		Context("when cleanup fails", func() {
			BeforeEach(func() {
				remoteRunner.RemoveDirectoryReturns(fmt.Errorf("fool!"))
			})

			It("returns an error", func() {
				Expect(err).To(SatisfyAll(
					MatchError(ContainSubstring("Unable to clean up backup artifact")),
					MatchError(ContainSubstring("fool!")),
				))
			})
		})
	})
})

func createTempFile(contents string) string {
	tempFile, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())
	tempFile.Write([]byte(contents))
	tempFile.Close()
	return tempFile.Name()
}
