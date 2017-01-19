package bosh_test

import (
	"fmt"
	"log"

	"bytes"
	"io"

	"github.com/cloudfoundry/bosh-cli/director"
	boshfakes "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
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
	var boshLogger boshlog.Logger
	var boshDeployment *boshfakes.FakeDeployment
	var sshConnection *fakes.FakeSSHConnection

	var deploymentName = "kubernetes"

	var stdoutLogStream *bytes.Buffer
	var stderrLogStream *bytes.Buffer

	var b backuper.BoshDirector
	JustBeforeEach(func() {
		b = bosh.New(boshDirector, optsGenerator.Spy, sshConnectionFactory.Spy, boshLogger)
	})

	BeforeEach(func() {
		optsGenerator = new(fakes.FakeSSHOptsGenerator)
		sshConnectionFactory = new(fakes.FakeSSHConnectionFactory)
		boshDirector = new(boshfakes.FakeDirector)
		boshDeployment = new(boshfakes.FakeDeployment)
		sshConnection = new(fakes.FakeSSHConnection)

		stdoutLogStream = bytes.NewBufferString("")
		stderrLogStream = bytes.NewBufferString("")

		combinecOutLog := log.New(io.MultiWriter(GinkgoWriter, stdoutLogStream), "[bosh-package] ", log.Lshortfile)
		combinedErrLog := log.New(io.MultiWriter(GinkgoWriter, stderrLogStream), "[bosh-package] ", log.Lshortfile)
		boshLogger = boshlog.New(boshlog.LevelDebug, combinecOutLog, combinedErrLog)
	})
	Describe("FindInstances", func() {
		var stubbedSshOpts director.SSHOpts = director.SSHOpts{Username: "user"}
		var actualInstances []backuper.Instance
		var acutalError error
		JustBeforeEach(func() {
			actualInstances, acutalError = b.FindInstances(deploymentName)
		})

		Context("finds instances for the deployment", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{{
					JobName: "job1",
					ID:      "jobID",
				}}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:  "username",
						Host:      "hostname",
						IndexOrID: "jobID",
					},
				}}, nil)
				sshConnectionFactory.Returns(sshConnection, nil)
				sshConnection.RunReturns([]byte("/var/vcap/jobs/consul_agent/bin/p-backup\n"+
					"/var/vcap/jobs/metron_agent/bin/p-backup"), nil, 0, nil)
			})
			It("collects the instances", func() {
				Expect(actualInstances).To(Equal([]backuper.Instance{bosh.NewBoshInstance("job1",
					"0",
					"jobID",
					sshConnection,
					boshDeployment,
					boshLogger,
					bosh.BackupAndRestoreScripts{
						"/var/vcap/jobs/consul_agent/bin/p-backup",
						"/var/vcap/jobs/metron_agent/bin/p-backup",
					},
				)}))
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
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job1", "")))
				Expect(opts).To(Equal(stubbedSshOpts))
			})

			It("creates a ssh connection to each host", func() {
				Expect(sshConnectionFactory.CallCount()).To(Equal(1))
				host, username, privateKey := sshConnectionFactory.ArgsForCall(0)
				Expect(host).To(Equal("hostname:22"), "overrides the port to be 22 if not provided")
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
			})
			It("finds the scripts on each host", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/* -type f"))
			})
		})

		Context("finds instances with no jobs folder", func() {
			var (
				actualStdOut = "stdout"
				actualStdErr = "find: `/var/vcap/jobs/*/bin/*': No such file or directory"
			)

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
				sshConnection.RunReturns([]byte(actualStdOut), []byte(actualStdErr), 1, nil)
			})

			It("does not fail", func() {
				Expect(acutalError).NotTo(HaveOccurred())
			})

			It("collects the instances, with no scripts", func() {
				Expect(actualInstances).To(Equal([]backuper.Instance{bosh.NewBoshInstance("job1",
					"0",
					"index",
					sshConnection,
					boshDeployment,
					boshLogger,
					bosh.BackupAndRestoreScripts{},
				)}))
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

			It("uses the ssh connnection to run find", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
			})

			It("logs the stdout", func() {
				Expect(stdoutLogStream.String()).To(ContainSubstring(actualStdOut))

			})
			It("logs the stderr", func() {
				Expect(stdoutLogStream.String()).To(ContainSubstring(actualStdErr))

			})
		})
		Context("finds instances for the deployment, with port specified in host", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{{
					JobName: "job1",
				}}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:  "username",
						Host:      "hostname:3457",
						IndexOrID: "index",
					},
				}}, nil)
				sshConnectionFactory.Returns(sshConnection, nil)
			})

			It("uses the specified port", func() {
				Expect(sshConnectionFactory.CallCount()).To(Equal(1))
				host, username, privateKey := sshConnectionFactory.ArgsForCall(0)
				Expect(host).To(Equal("hostname:3457"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
			})
		})

		Context("finds instances for the deployment, having multiple instances in an instance group", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{
					{
						JobName: "job1",
						ID:      "id1",
					},
					{
						JobName: "job1",
						ID:      "id2",
					},
				}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:  "username",
						Host:      "hostname1",
						IndexOrID: "id1",
					},
					{
						Username:  "username",
						Host:      "hostname2",
						IndexOrID: "id2",
					},
				}}, nil)
				sshConnectionFactory.Returns(sshConnection, nil)
				sshConnection.RunReturns([]byte("/var/vcap/jobs/consul_agent/bin/p-backup"), nil, 0, nil)
			})
			It("collects the instances", func() {
				Expect(actualInstances).To(Equal([]backuper.Instance{
					bosh.NewBoshInstance("job1", "0", "id1", sshConnection, boshDeployment, boshLogger, bosh.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/p-backup"}),
					bosh.NewBoshInstance("job1", "1", "id2", sshConnection, boshDeployment, boshLogger, bosh.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/p-backup"}),
				}))
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
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job1", "")))
				Expect(opts).To(Equal(stubbedSshOpts))
			})

			It("creates a ssh connection to each host", func() {
				Expect(sshConnectionFactory.CallCount()).To(Equal(2))

				host, username, privateKey := sshConnectionFactory.ArgsForCall(0)
				Expect(host).To(Equal("hostname1:22"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))

				host, username, privateKey = sshConnectionFactory.ArgsForCall(1)
				Expect(host).To(Equal("hostname2:22"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
			})

			It("finds the scripts on each host", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(2))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/* -type f"))
				Expect(sshConnection.RunArgsForCall(1)).To(Equal("find /var/vcap/jobs/*/bin/* -type f"))
			})
		})

		//TODO: multiple instance groups
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

			Context("we cant get scripts information", func() {
				var (
					actualStdOut = "stdout"
					actualStdErr = "stderr"
				)

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
					sshConnection.RunReturns([]byte(actualStdOut), []byte(actualStdErr), 0, expectedError)
				})

				It("does fail", func() {
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

				It("uses the ssh connnection to run find", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})
				It("logs the failure", func() {
					Expect(stderrLogStream.String()).To(ContainSubstring(expectedError.Error()))

				})
				It("logs the stdout", func() {
					Expect(stderrLogStream.String()).To(ContainSubstring(actualStdOut))

				})
				It("logs the stderr", func() {
					Expect(stderrLogStream.String()).To(ContainSubstring(actualStdErr))

				})
			})

			Context("find fails with an unknown error", func() {
				var (
					actualStdOut = "stdout"
					actualStdErr = "unknown error"
				)

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
					sshConnection.RunReturns([]byte(actualStdOut), []byte(actualStdErr), 1, nil)
				})

				It("does fail", func() {
					Expect(acutalError).To(HaveOccurred())
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

				It("uses the ssh connnection to run find", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})

				It("logs the stdout", func() {
					Expect(stderrLogStream.String()).To(ContainSubstring(actualStdOut))

				})
				It("logs the stderr", func() {
					Expect(stderrLogStream.String()).To(ContainSubstring(actualStdErr))

				})
			})
		})
	})

	Describe("GetManifest", func() {
		var actualManifest string
		var acutalError error
		JustBeforeEach(func() {
			actualManifest, acutalError = b.GetManifest(deploymentName)
		})

		Context("gets the manifest", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.ManifestReturns("a good ol manifest", nil)
			})
			It("from the deployment", func() {
				Expect(actualManifest).To(Equal("a good ol manifest"))
			})
		})
		Context("fails", func() {
			Context("to find deployment", func() {
				var findDeploymentError = fmt.Errorf("what do you have to loose?")
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(nil, findDeploymentError)
				})
				It("returns an error", func() {
					Expect(acutalError).To(MatchError(findDeploymentError))
				})
			})
			Context("to download manifest", func() {
				var downloadManifestError = fmt.Errorf("you will be tired of winning")
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.ManifestReturns("", downloadManifestError)
				})
				It("returns an error", func() {
					Expect(acutalError).To(MatchError(downloadManifestError))
				})
			})
		})
	})
})
