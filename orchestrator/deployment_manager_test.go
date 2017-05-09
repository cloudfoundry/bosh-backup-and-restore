package orchestrator_test

import (
	"fmt"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("DeploymentManager", func() {
	var boshClient *fakes.FakeBoshClient
	var logger *fakes.FakeLogger
	var deploymentName = "brownie"
	var artifact *fakes.FakeArtifact
	var manifest string

	var deploymentManager orchestrator.DeploymentManager
	BeforeEach(func() {
		boshClient = new(fakes.FakeBoshClient)
		logger = new(fakes.FakeLogger)
	})
	JustBeforeEach(func() {
		deploymentManager = orchestrator.NewBoshDeploymentManager(boshClient, logger)
	})

	Context("Find", func() {
		var findError error
		var deployment orchestrator.Deployment
		var instances []orchestrator.Instance
		BeforeEach(func() {
			instances = []orchestrator.Instance{new(fakes.FakeInstance)}
			boshClient.FindInstancesReturns(instances, nil)
		})
		JustBeforeEach(func() {
			deployment, findError = deploymentManager.Find(deploymentName)
		})
		It("asks the bosh director for instances", func() {
			Expect(boshClient.FindInstancesCallCount()).To(Equal(1))
			Expect(boshClient.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
		})
		It("returns the deployment manager with instances", func() {
			Expect(deployment).To(Equal(orchestrator.NewBoshDeployment(logger, instances)))
		})

		Context("error finding instances", func() {
			var expectedFindError = fmt.Errorf("a tuna sandwich")
			BeforeEach(func() {
				boshClient.FindInstancesReturns(nil, expectedFindError)
			})

			It("returns an error", func() {
				Expect(findError).To(MatchError(expectedFindError))
			})
		})
	})

	Describe("SaveManifest", func() {
		var saveManifestError error
		JustBeforeEach(func() {
			saveManifestError = deploymentManager.SaveManifest(deploymentName, artifact)
		})

		Context("successfully saves the manifest", func() {
			BeforeEach(func() {
				artifact = new(fakes.FakeArtifact)
				manifest = "foo"
				boshClient.GetManifestReturns(manifest, nil)
			})

			It("asks the bosh director for the manifest", func() {
				Expect(boshClient.GetManifestCallCount()).To(Equal(1))
				Expect(boshClient.GetManifestArgsForCall(0)).To(Equal(deploymentName))
			})

			It("saves the manifest to the artifact", func() {
				Expect(artifact.SaveManifestCallCount()).To(Equal(1))
				Expect(artifact.SaveManifestArgsForCall(0)).To(Equal(manifest))
			})
			It("should succeed", func() {
				Expect(saveManifestError).To(Succeed())
			})
		})

		Context("fails to fetch the manifest", func() {
			var manifestFetchError = fmt.Errorf("Boring error")
			BeforeEach(func() {
				artifact = new(fakes.FakeArtifact)
				boshClient.GetManifestReturns("", manifestFetchError)
			})

			It("asks the bosh director for the manifest", func() {
				Expect(boshClient.GetManifestCallCount()).To(Equal(1))
				Expect(boshClient.GetManifestArgsForCall(0)).To(Equal(deploymentName))
			})

			It("does not save the manifest to the artifact", func() {
				Expect(artifact.SaveManifestCallCount()).To(BeZero())
			})

			It("should fail", func() {
				Expect(saveManifestError).To(MatchError(manifestFetchError))
			})
		})

		Context("fails to save the manifest", func() {
			var manifestSaveError = fmt.Errorf("Boring")

			BeforeEach(func() {
				artifact = new(fakes.FakeArtifact)
				boshClient.GetManifestReturns(manifest, nil)
				artifact.SaveManifestReturns(manifestSaveError)
			})

			It("asks the bosh director for the manifest", func() {
				Expect(boshClient.GetManifestCallCount()).To(Equal(1))
				Expect(boshClient.GetManifestArgsForCall(0)).To(Equal(deploymentName))
			})

			It("saves the manifest to the artifact", func() {
				Expect(artifact.SaveManifestCallCount()).To(Equal(1))
				Expect(artifact.SaveManifestArgsForCall(0)).To(Equal(manifest))
			})

			It("should fail", func() {
				Expect(saveManifestError).To(MatchError(manifestSaveError))
			})
		})
	})

})
