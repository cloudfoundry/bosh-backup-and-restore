package standalone_test

import (
	"fmt"
	"os"

	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	. "github.com/pivotal-cf/bosh-backup-and-restore/standalone"

	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	instancefakes "github.com/pivotal-cf/bosh-backup-and-restore/instance/fakes"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator/fakes"
	sshfakes "github.com/pivotal-cf/bosh-backup-and-restore/ssh/fakes"
)

var _ = Describe("DeploymentManager", func() {
	var deploymentManager DeploymentManager
	var deploymentName = "director"
	var artifact *fakes.FakeArtifact
	var logger *fakes.FakeLogger
	var hostName = "hostname"
	var username = "username"
	var privateKey string
	var fakeJobFinder *instancefakes.FakeJobFinder
	var fakeConnFactory *sshfakes.FakeSSHConnectionFactory
	var fakeSSHConnection *sshfakes.FakeSSHConnection

	BeforeEach(func() {
		privateKey = createTempFile("privateKey")
		logger = new(fakes.FakeLogger)
		artifact = new(fakes.FakeArtifact)
		fakeConnFactory = new(sshfakes.FakeSSHConnectionFactory)
		fakeJobFinder = new(instancefakes.FakeJobFinder)
		fakeSSHConnection = new(sshfakes.FakeSSHConnection)

		deploymentManager = NewDeploymentManager(logger, hostName, username, privateKey, fakeJobFinder, fakeConnFactory.Spy)
	})

	AfterEach(func() {
		os.Remove(privateKey)
	})

	Describe("Find", func() {
		var actualDeployment orchestrator.Deployment
		var actualError error
		var fakeJobs instance.Jobs

		JustBeforeEach(func() {
			actualDeployment, actualError = deploymentManager.Find(deploymentName)
		})

		Context("success", func() {
			BeforeEach(func() {
				fakeJobs = instance.Jobs{instance.NewJob(instance.BackupAndRestoreScripts{"foo"}, instance.Metadata{})}
				fakeConnFactory.Returns(fakeSSHConnection, nil)
				fakeJobFinder.FindJobsReturns(fakeJobs, nil)
			})
			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("invokes connection creator", func() {
				Expect(fakeConnFactory.CallCount()).To(Equal(1))
			})

			It("invokes job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(1))
			})

			It("returns a deployment", func() {
				Expect(actualDeployment).To(Equal(orchestrator.NewDeployment(logger, []orchestrator.Instance{
					instance.NewDeployedInstance("0", "director", "0", fakeSSHConnection, logger, fakeJobs),
				})))
			})
		})

		Context("can't read private key", func() {
			BeforeEach(func() {
				os.Remove(privateKey)
			})

			It("should fail", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("should not invoke connection creator", func() {
				Expect(fakeConnFactory.CallCount()).To(BeZero())
			})
		})

		Context("can't create SSH connection", func() {
			connError := fmt.Errorf("error")

			BeforeEach(func() {
				fakeConnFactory.Returns(nil, connError)
			})

			It("should fail", func() {
				Expect(actualError).To(MatchError(connError))
			})

			It("should invoke connection creator", func() {
				Expect(fakeConnFactory.CallCount()).To(Equal(1))
			})

			It("should not invoke job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(BeZero())
			})

		})

		Context("can't find jobs", func() {
			findJobsErr := fmt.Errorf("error")

			BeforeEach(func() {
				fakeConnFactory.Returns(fakeSSHConnection, nil)
				fakeJobFinder.FindJobsReturns(nil, findJobsErr)
			})

			It("should fail", func() {
				Expect(actualError).To(MatchError(findJobsErr))
			})

			It("should invoke connection creator", func() {
				Expect(fakeConnFactory.CallCount()).To(Equal(1))
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

func createTempFile(contents string) string {
	tempFile, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())
	tempFile.Write([]byte(contents))
	tempFile.Close()
	return tempFile.Name()
}
