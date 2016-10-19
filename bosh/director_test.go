package bosh_test

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cli/director"
	boshfakes "github.com/cloudfoundry/bosh-cli/director/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh/fakes"
)

var _ = Describe("Director", func() {
	var optsGenerator *fakes.FakeSSHOptsGenerator
	var sshConnectionFactory *fakes.FakeSSHConnectionFactory
	var boshDirector *boshfakes.FakeDirector
	var boshDeployment *boshfakes.FakeDeployment
	var sshConnection *fakes.FakeSSHConnection

	var deploymentName = "kubernetes"

	var b backuper.BoshDirector
	JustBeforeEach(func() {
		b = bosh.New(boshDirector, optsGenerator.Spy, sshConnectionFactory.Spy)
	})

	BeforeEach(func() {
		optsGenerator = new(fakes.FakeSSHOptsGenerator)
		sshConnectionFactory = new(fakes.FakeSSHConnectionFactory)
		boshDirector = new(boshfakes.FakeDirector)
		boshDeployment = new(boshfakes.FakeDeployment)
		sshConnection = new(fakes.FakeSSHConnection)
	})
	Context("instances", func() {
		var stubbedSshOpts director.SSHOpts = director.SSHOpts{Username: "user"}
		var acutalInstances backuper.Instances
		var acutalError error
		JustBeforeEach(func() {
			acutalInstances, acutalError = b.FindInstances(deploymentName)
		})

		Context("finds instances for the deployment", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{{
					JobName: "job1",
				}}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:  "username",
						Host:      "hostname",
						IndexOrID: "index",
					},
				}}, nil)
				sshConnectionFactory.Returns(sshConnection, nil)
			})
			It("collects the instances", func() {
				Expect(acutalInstances).To(Equal(backuper.Instances{bosh.NewBoshInstance("job1", "index", sshConnection)}))
			})
			It("does not fail", func() {
				Expect(acutalError).NotTo(HaveOccurred())
			})

			It("fetches the deployment by name", func() {
				Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
				Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
			})

			It("fetchs vms for the deployment", func() {
				Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
			})

			It("generates a new ssh private key", func() {
				Expect(optsGenerator.CallCount()).To(Equal(1))
			})

			It("sets up ssh for each group found", func() {
				Expect(boshDeployment.SetUpSSHCallCount()).To(Equal(1))

				slug, opts := boshDeployment.SetUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrPoolOrInstanceSlug("job1", "")))
				Expect(opts).To(Equal(stubbedSshOpts))
			})

			It("creates a ssh connection to each host", func() {
				Expect(sshConnectionFactory.CallCount()).To(Equal(1))
				host, username, privateKey := sshConnectionFactory.ArgsForCall(0)
				Expect(host).To(Equal("hostname"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
			})
		})

		//TODO: multiple instance groups
		//TODO: multiple vms in instance groups
		Context("failures", func() {
			var expectedError = fmt.Errorf("er ma gerd")

			Context("fails to find the deployment", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(nil, expectedError)
				})

				It("does fails", func() {
					Expect(acutalError).To(MatchError(expectedError))
				})

				It("tries to fetch deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
			})

			Context("fails to find vms for a deployment", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns(nil, expectedError)
				})

				It("does fails", func() {
					Expect(acutalError).To(MatchError(expectedError))
				})
				It("tries to fetch vm infos", func() {
					Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
				})

				It("fetches deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
			})
			Context("fails to generate ssh opts", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)

					optsGenerator.Returns(director.SSHOpts{}, "", expectedError)
				})
				It("does fails", func() {
					Expect(acutalError).To(MatchError(expectedError))
				})

				It("tries to generate ssh keys", func() {
					Expect(optsGenerator.CallCount()).To(Equal(1))
				})
			})

			Context("fails if a invalid job name is received", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{{
						JobName: "this/is/invalid",
					}}, nil)
				})
				It("does fails", func() {
					Expect(acutalError).To(HaveOccurred())
				})

				It("tries to fetch deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})

				It("fetchs vms for the deployment", func() {
					Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
				})
			})

			Context("fails while setting up ssh, on the vm", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{{
						JobName: "job1",
					}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
					boshDeployment.SetUpSSHReturns(director.SSHResult{}, expectedError)
				})

				It("does fails", func() {
					Expect(acutalError).To(MatchError(expectedError))
				})

				It("tries to fetch vm infos", func() {
					Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
				})

				It("fetches deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
				It("generates ssh opts", func() {
					Expect(optsGenerator.CallCount()).To(Equal(1))
				})
			})

			Context("fails creating a ssh connection, to the vm", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{{
						JobName: "job1",
					}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
					boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
						{
							Username:  "username",
							Host:      "hostname",
							IndexOrID: "index",
						},
					}}, nil)
					sshConnectionFactory.Returns(nil, expectedError)
				})

				It("does fails", func() {
					Expect(acutalError).To(MatchError(expectedError))
				})

				It("tries to connect to the vm", func() {
					Expect(sshConnectionFactory.CallCount()).To(Equal(1))
				})

				It("fetchs vm infos", func() {
					Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
				})

				It("fetches deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
				It("generates ssh opts", func() {
					Expect(optsGenerator.CallCount()).To(Equal(1))
				})
			})
		})
	})
})
