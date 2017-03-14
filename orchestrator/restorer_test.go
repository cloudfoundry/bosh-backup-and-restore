package orchestrator_test

import (
	"fmt"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("restorer", func() {
	Context("restores a deployment from backup", func() {
		var (
			restoreError      error
			artifactManager   *fakes.FakeArtifactManager
			artifact          *fakes.FakeArtifact
			boshClient        *fakes.FakeBoshClient
			logger            *fakes.FakeLogger
			instances         []orchestrator.Instance
			b                 *orchestrator.Restorer
			deploymentName    string
			deploymentManager *fakes.FakeDeploymentManager
			deployment        *fakes.FakeDeployment
		)

		BeforeEach(func() {
			instances = []orchestrator.Instance{new(fakes.FakeInstance)}
			boshClient = new(fakes.FakeBoshClient)
			logger = new(fakes.FakeLogger)
			artifactManager = new(fakes.FakeArtifactManager)
			artifact = new(fakes.FakeArtifact)
			deploymentManager = new(fakes.FakeDeploymentManager)
			deployment = new(fakes.FakeDeployment)

			artifactManager.OpenReturns(artifact, nil)
			deploymentManager.FindReturns(deployment, nil)
			deployment.IsRestorableReturns(true)
			deployment.InstancesReturns(instances)
			artifact.DeploymentMatchesReturns(true, nil)
			artifact.ValidReturns(true, nil)

			b = orchestrator.NewRestorer(boshClient, artifactManager, logger, deploymentManager)

			deploymentName = "deployment-to-restore"
		})

		JustBeforeEach(func() {
			restoreError = b.Restore(deploymentName)
		})

		It("does not fail", func() {
			Expect(restoreError).NotTo(HaveOccurred())
		})

		It("ensures that instance is cleaned up", func() {
			Expect(deployment.CleanupCallCount()).To(Equal(1))
		})

		It("ensures artifact is valid", func() {
			Expect(artifact.ValidCallCount()).To(Equal(1))
		})

		It("finds the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the deployment is restorable", func() {
			Expect(deployment.IsRestorableCallCount()).To(Equal(1))
		})

		It("checks that the deployment topology matches the topology of the backup", func() {
			Expect(artifactManager.OpenCallCount()).To(Equal(1))
			Expect(artifact.DeploymentMatchesCallCount()).To(Equal(1))

			name, actualInstances := artifact.DeploymentMatchesArgsForCall(0)
			Expect(name).To(Equal(deploymentName))
			Expect(actualInstances).To(Equal(instances))
		})

		It("streams the local backup to the deployment", func() {
			Expect(deployment.CopyLocalBackupToRemoteCallCount()).To(Equal(1))
			Expect(deployment.CopyLocalBackupToRemoteArgsForCall(0)).To(Equal(artifact))
		})

		It("calls restore on the deployment", func() {
			Expect(deployment.RestoreCallCount()).To(Equal(1))
		})

		Describe("failures", func() {

			var assertCleanupError = func() {
				var cleanupError = fmt.Errorf("he was born in kenya")
				BeforeEach(func() {
					deployment.CleanupReturns(cleanupError)
				})

				It("includes the cleanup error in the returned error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring(cleanupError.Error())))
				})
			}

			Context("fails to find deployment", func() {
				BeforeEach(func() {
					deploymentManager.FindReturns(nil, fmt.Errorf("they will pay for the wall"))
				})

				It("returns an error", func() {
					Expect(restoreError).To(MatchError("they will pay for the wall"))
				})
			})

			Context("fails if the artifact cant be opened", func() {
				var artifactOpenError = fmt.Errorf("i have the best brain")
				BeforeEach(func() {
					deploymentManager.FindReturns(deployment, nil)
					artifactManager.OpenReturns(nil, artifactOpenError)
				})
				It("returns an error", func() {
					Expect(restoreError).To(MatchError(artifactOpenError))
				})
			})
			Context("fails if the artifact is invalid", func() {
				BeforeEach(func() {
					deploymentManager.FindReturns(deployment, nil)
					artifactManager.OpenReturns(artifact, nil)
					artifact.ValidReturns(false, nil)
				})
				It("returns an error", func() {
					Expect(restoreError).To(MatchError("Backup artifact is corrupted"))
				})
			})

			Context("fails, if the cleanup fails", func() {
				var cleanupError = fmt.Errorf("we gotta deal with china")
				BeforeEach(func() {
					deploymentManager.FindReturns(deployment, nil)
					artifactManager.OpenReturns(artifact, nil)
					artifact.ValidReturns(true, nil)
					deployment.CleanupReturns(cleanupError)
				})

				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring(cleanupError.Error())))
				})

				It("returns an error of type, cleanup error", func() {
					Expect(restoreError).To(BeAssignableToTypeOf(orchestrator.CleanupError{}))
				})
			})
			Context("fails if can't check if artifact is valid", func() {
				var artifactValidError = fmt.Errorf("we will win so much")

				BeforeEach(func() {
					deploymentManager.FindReturns(deployment, nil)
					artifactManager.OpenReturns(artifact, nil)
					artifact.ValidReturns(false, artifactValidError)
				})
				It("returns an error", func() {
					Expect(restoreError).To(MatchError(artifactValidError))
				})
			})

			Context("if deployment not restorable", func() {
				BeforeEach(func() {
					deployment.IsRestorableReturns(false)
				})

				It("returns an error", func() {
					Expect(restoreError).To(HaveOccurred())
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
				assertCleanupError()
			})

			Context("if the deployment's topology doesn't match that of the backup", func() {
				BeforeEach(func() {
					artifact.DeploymentMatchesReturns(false, nil)
				})

				It("returns an error", func() {
					Expect(restoreError).To(HaveOccurred())
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
				assertCleanupError()
			})

			Context("if checking the deployment topology fails", func() {
				BeforeEach(func() {
					artifact.DeploymentMatchesReturns(true, fmt.Errorf("my fingers are long and beautiful"))
				})

				It("returns an error", func() {
					Expect(restoreError).To(HaveOccurred())
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
				assertCleanupError()
			})

			Context("if streaming the backup to the remote fails", func() {
				BeforeEach(func() {
					deployment.CopyLocalBackupToRemoteReturns(fmt.Errorf("Broken pipe"))
				})

				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring("Unable to send backup to remote machine. Got error: Broken pipe")))
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
				assertCleanupError()
			})

			Context("if running the restore script fails", func() {
				var restoreError = fmt.Errorf("there is something in that birth certificate")
				BeforeEach(func() {
					deployment.RestoreReturns(restoreError)
				})

				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring(restoreError.Error())))
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
				assertCleanupError()
			})
		})
	})
})
