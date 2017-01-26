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
	"strings"
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
		var (
			findScriptsSshStdout,
			findScriptsSshStderr,
			lsMetadataSshStdout,
			lsMetadataSshStderr,
			runMetadataSshStdout,
			runMetadataSshStderr []byte

			findScriptsExitCode, lsMetadataExitCode, runMetadataExitCode int
			stubbedSshOpts director.SSHOpts = director.SSHOpts{Username: "user"}
			actualInstances []backuper.Instance
			actualError, findScriptsError, lsMetadataError, runMetadataError error
		)

		JustBeforeEach(func() {
			sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
				switch {
				case strings.HasPrefix(cmd, "find "):
					return findScriptsSshStdout, findScriptsSshStderr, findScriptsExitCode, findScriptsError
				case strings.HasPrefix(cmd, "ls "):
					return lsMetadataSshStdout, lsMetadataSshStderr, lsMetadataExitCode, lsMetadataError
				case strings.HasPrefix(cmd, "/var/vcap/jobs/"):
					return runMetadataSshStdout, runMetadataSshStderr, runMetadataExitCode, runMetadataError
				}

				return nil, nil, 0, nil
			}

			actualInstances, actualError = b.FindInstances(deploymentName)
		})

		BeforeEach(func() {
			findScriptsSshStdout = []byte{}
			findScriptsSshStderr = []byte{}

			lsMetadataSshStdout = []byte{}
			lsMetadataSshStderr = []byte("No such file or directory")
			lsMetadataExitCode = 1
			lsMetadataError = nil

			runMetadataSshStdout = []byte{}
			runMetadataError = nil
			runMetadataExitCode = 0

			findScriptsExitCode = 0
			findScriptsError = nil
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
				findScriptsSshStdout = []byte("/var/vcap/jobs/consul_agent/bin/p-backup\n"+
					"/var/vcap/jobs/consul_agent/bin/p-restore")
			})

			It("collects the instances", func() {
				jobs, _ := bosh.NewJobs(bosh.BackupAndRestoreScripts{
					"/var/vcap/jobs/consul_agent/bin/p-backup",
					"/var/vcap/jobs/consul_agent/bin/p-restore",
				}, map[string]string{})

				Expect(actualInstances).To(Equal([]backuper.Instance{bosh.NewBoshInstance(
					"job1",
					"0",
					"jobID",
					sshConnection,
					boshDeployment,
					boshLogger,
					jobs,
				)}))
			})

			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
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
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/* -type f"))
			})

			Context("when job does not specify a custom artifact name", func() {
				BeforeEach(func() {
					lsMetadataSshStderr = []byte("No such file or directory")
					lsMetadataExitCode = 1
				})

				It("does not fail", func() {
					Expect(actualError).NotTo(HaveOccurred())
				})
			})
			
			Context("when job specifies a custom artifact name", func() {
				BeforeEach(func() {
					lsMetadataSshStdout = []byte("/var/vcap/jobs/consul_agent/bin/p-metadata")
					runMetadataSshStdout = []byte(`---
backup_name: consul_backup`)
				})

				It("collects the instances with the custom artifact name", func() {
					metadata := map[string]string{
						"consul_agent": "consul_backup",
					}

					jobs, _ := bosh.NewJobs(bosh.BackupAndRestoreScripts{
						"/var/vcap/jobs/consul_agent/bin/p-backup",
						"/var/vcap/jobs/consul_agent/bin/p-restore",
					}, metadata)

					Expect(actualInstances).To(Equal([]backuper.Instance{bosh.NewBoshInstance(
						"job1",
						"0",
						"jobID",
						sshConnection,
						boshDeployment,
						boshLogger,
						jobs,
					)}))
				})

				Context("when the metadata YAML is malformed", func() {
					BeforeEach(func() {
						runMetadataSshStdout = []byte(`this-is-terrible`)
					})

					It("fails", func() {
						Expect(actualError).To(HaveOccurred())
					})

					It("returns an error with an explanation of the failure", func() {
						Expect(actualError.Error()).To(ContainSubstring(
							"Reading job metadata for hostname/jobID failed",
						))
					})

					It("logs an error message", func() {
						Expect(stderrLogStream.String()).To(ContainSubstring(
							"Reading job metadata for hostname/jobID failed",
						))
					})
				})

				Context("when listing metadata scripts fails", func() {
					BeforeEach(func() {
						lsMetadataSshStdout = []byte("some stdout")
						lsMetadataSshStderr = []byte("some stderr")
						lsMetadataExitCode = 1
					})

					It("fails", func() {
						Expect(actualError).To(HaveOccurred())
					})

					It("returns an error with an explanation of the failure and the stdout and stderr", func() {
						Expect(actualError.Error()).To(ContainSubstring(
							"Failed to check for job metadata scripts on hostname/jobID",
						))

						Expect(actualError.Error()).To(ContainSubstring(
							fmt.Sprintf("Stdout: %s", string(lsMetadataSshStdout)),
						))

						Expect(actualError.Error()).To(ContainSubstring(
							fmt.Sprintf("Stderr: %s", string(lsMetadataSshStderr)),
						))
					})
				})

				Context("when an SSH error occurs while listing metadata scripts fails", func(){
					BeforeEach(func() {
						lsMetadataError = fmt.Errorf("this is a boring error")
					})

					It("fails", func() {
						Expect(actualError).To(HaveOccurred())
					})

					It("returns an error with an explanation of the failure", func() {
						Expect(actualError.Error()).To(ContainSubstring(
							"An error occurred while checking for job metadata scripts on hostname/jobID",
						))
					})

					It("logs an error message", func() {
						Expect(stderrLogStream.String()).To(ContainSubstring(
							"An error occurred while checking for job metadata scripts on hostname/jobID",
						))
					})
				})

				Context("when running a metadata script fails", func() {
					BeforeEach(func() {
						runMetadataSshStdout = []byte("some stdout")
						runMetadataSshStderr = []byte("It went very wrong")
						runMetadataExitCode = 1
					})

					It("fails", func() {
						Expect(actualError).To(HaveOccurred())
					})

					It("returns an error with an explanation of the failure", func() {
						Expect(actualError.Error()).To(ContainSubstring(
							"Failed to run job metadata scripts on hostname/jobID",
						))
					})
				})

				Context("when an SSH error occurs while running metadata scripts", func(){
					BeforeEach(func() {
						runMetadataError = fmt.Errorf("everything is awful")
					})

					It("fails", func() {
						Expect(actualError).To(HaveOccurred())
					})

					It("returns an error with an explanation of the failure", func() {
						Expect(actualError.Error()).To(ContainSubstring(
							"An error occurred while running job metadata scripts on hostname/jobID",
						))
					})

					It("logs an error message", func() {
						Expect(stderrLogStream.String()).To(ContainSubstring(
							"An error occurred while running job metadata scripts on hostname/jobID",
						))
					})
				})
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
				findScriptsSshStdout = []byte(actualStdOut)
				findScriptsSshStderr = []byte(actualStdErr)
				findScriptsExitCode = 1
			})

			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("collects the instances, with no scripts", func() {
				jobs, _ := bosh.NewJobs(bosh.BackupAndRestoreScripts{}, map[string]string{})

				Expect(actualInstances).To(Equal([]backuper.Instance{bosh.NewBoshInstance("job1",
					"0",
					"index",
					sshConnection,
					boshDeployment,
					boshLogger,
					jobs,
				)}))
			})

			It("tries to connect to the vm", func() {
				Expect(sshConnectionFactory.CallCount()).To(Equal(1))
			})

			It("fetches vm infos", func() {
				Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
			})

			It("fetches deployment", func() {
				Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
				Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
			})

			It("generates ssh opts", func() {
				Expect(optsGenerator.CallCount()).To(Equal(1))
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
				findScriptsSshStdout = []byte("/var/vcap/jobs/consul_agent/bin/p-backup")
			})
			It("collects the instances", func() {
				instance0Jobs, _ := bosh.NewJobs(
					bosh.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/p-backup"},
					map[string]string{},
				)

				instance1Jobs, _ := bosh.NewJobs(
					bosh.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/p-backup"},
					map[string]string{},
				)

				Expect(actualInstances).To(Equal([]backuper.Instance{
					bosh.NewBoshInstance(
						"job1",
						"0",
						"id1",
						sshConnection,
						boshDeployment,
						boshLogger,
						instance0Jobs,
					),
					bosh.NewBoshInstance(
						"job1",
						"1",
						"id2",
						sshConnection,
						boshDeployment,
						boshLogger,
						instance1Jobs,
					),
				}))
			})
			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
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
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/* -type f"))
				Expect(sshConnection.RunArgsForCall(3)).To(Equal("find /var/vcap/jobs/*/bin/* -type f"))
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
					Expect(actualError).To(MatchError(expectedError))
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
					Expect(actualError).To(MatchError(expectedError))
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
					Expect(actualError).To(MatchError(expectedError))
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
					Expect(actualError).To(HaveOccurred())
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
					Expect(actualError).To(MatchError(expectedError))
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
					Expect(actualError).To(MatchError(expectedError))
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
					findScriptsSshStdout = []byte(actualStdOut)
					findScriptsSshStderr = []byte(actualStdErr)
					findScriptsError = expectedError
				})

				It("does fail", func() {
					Expect(actualError).To(MatchError(expectedError))
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
					findScriptsSshStdout = []byte(actualStdOut)
					findScriptsSshStderr = []byte(actualStdErr)
					findScriptsExitCode = 1
				})

				It("does fail", func() {
					Expect(actualError).To(HaveOccurred())
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
