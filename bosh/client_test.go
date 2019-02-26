package bosh_test

import (
	"log"

	"bytes"
	"io"

	"errors"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	instancefakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance/fakes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	sshfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
	"github.com/cloudfoundry/bosh-cli/director"
	boshfakes "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gossh "golang.org/x/crypto/ssh"
)

var _ = Describe("Director", func() {
	var optsGenerator *sshfakes.FakeSSHOptsGenerator
	var remoteRunnerFactory *sshfakes.FakeRemoteRunnerFactory
	var boshDirector *boshfakes.FakeDirector
	var boshLogger boshlog.Logger
	var boshDeployment *boshfakes.FakeDeployment
	var remoteRunner *sshfakes.FakeRemoteRunner
	var fakeJobFinder *instancefakes.FakeJobFinder
	var manifestQuerierCreator *instancefakes.FakeManifestQuerierCreator
	var manifestQuerier *instancefakes.FakeManifestQuerier

	var deploymentName = "kubernetes"

	var logStream *bytes.Buffer

	var hostsPublicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAklOUpkDHrfHY17SbrmTIpNLTGK9Tjom/BWDSUGPl+nafzlHDTYW7hdI4yZ5ew18JH4JW9jbhUFrviQzM7xlELEVf4h9lFX5QVkbPppSwg0cda3Pbv7kOdJ/MTyBlWXFCR+HAo3FXRitBqxiX1nKhXpHAZsMciLq8V6RjsNAQwdsdMFvSlVK/7XAt3FaoJoAsncM1Q9x5+3V0Ww68/eIFmb1zuUFljQJKprrX88XypNDvjYNby6vw/Pb0rwert/EnmZ+AW4OZPnTPI89ZPmVMLuayrD2cE86Z/il8b+gw3r3+1nKatmIkjn2so1d01QraTlMqVSsbxNrRFi9wrf+M7Q== schacon@mylaptop.local"
	var hostKeyAlgorithm []string

	var b bosh.BoshClient

	JustBeforeEach(func() {
		b = bosh.NewClient(boshDirector, optsGenerator.Spy, remoteRunnerFactory.Spy, boshLogger, fakeJobFinder, manifestQuerierCreator.Spy)
	})

	BeforeEach(func() {
		optsGenerator = new(sshfakes.FakeSSHOptsGenerator)
		remoteRunnerFactory = new(sshfakes.FakeRemoteRunnerFactory)
		boshDirector = new(boshfakes.FakeDirector)
		boshDeployment = new(boshfakes.FakeDeployment)
		remoteRunner = new(sshfakes.FakeRemoteRunner)
		fakeJobFinder = new(instancefakes.FakeJobFinder)
		manifestQuerierCreator = new(instancefakes.FakeManifestQuerierCreator)
		manifestQuerier = new(instancefakes.FakeManifestQuerier)

		remoteRunner.IsWindowsReturns(false, nil)

		logStream = bytes.NewBufferString("")

		hostPublicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(hostsPublicKey))
		Expect(err).NotTo(HaveOccurred())
		hostKeyAlgorithm = []string{hostPublicKey.Type()}

		combinedLog := log.New(io.MultiWriter(GinkgoWriter, logStream), "[bosh-package] ", log.Lshortfile)
		boshLogger = boshlog.New(boshlog.LevelDebug, combinedLog)
	})

	Describe("FindInstances", func() {
		var (
			stubbedSshOpts  = director.SSHOpts{Username: "user"}
			actualInstances []orchestrator.Instance
			actualError     error
			expectedJobs    orchestrator.Jobs
		)

		JustBeforeEach(func() {
			actualInstances, actualError = b.FindInstances(deploymentName)
		})

		Context("finds instances for the deployment", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{{
					JobName: "job1",
					ID:      "jobID",
					Index:   newIndex(0),
				}}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:      "username",
						Host:          "10.0.0.0",
						IndexOrID:     "jobID",
						HostPublicKey: hostsPublicKey,
					},
				}}, nil)

				remoteRunnerFactory.Returns(remoteRunner, nil)
				expectedJobs = []orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						"",
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/consul_agent/bin/bbr/backup",
							"/var/vcap/jobs/consul_agent/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
				}
				fakeJobFinder.FindJobsReturns(expectedJobs, nil)

				manifestQuerierCreator.Returns(manifestQuerier, nil)
			})

			It("collects the instances", func() {
				Expect(actualInstances).To(Equal([]orchestrator.Instance{bosh.NewBoshDeployedInstance(
					"job1",
					"0",
					"jobID",
					remoteRunner,
					boshDeployment,
					false,
					boshLogger,
					expectedJobs,
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

			It("generates a manifest querier with the creator", func() {
				Expect(manifestQuerierCreator.CallCount()).To(Equal(1))
			})

			It("finds the jobs with the job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(1))
				_, _, manifestQuerier := fakeJobFinder.FindJobsArgsForCall(0)
				Expect(manifestQuerier).To(Equal(manifestQuerier))
			})

			It("sets up ssh for each group found", func() {
				Expect(boshDeployment.SetUpSSHCallCount()).To(Equal(1))

				slug, opts := boshDeployment.SetUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job1", "")))
				Expect(opts).To(Equal(stubbedSshOpts))
			})

			It("creates a remote runner for each host", func() {
				Expect(remoteRunnerFactory.CallCount()).To(Equal(1))
				host, username, privateKey, _, hostPublicKeyAlgorithm, logger := remoteRunnerFactory.ArgsForCall(0)
				Expect(host).To(Equal("10.0.0.0"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))
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
						Username:      "username",
						Host:          "10.0.0.0:3457",
						IndexOrID:     "index",
						HostPublicKey: hostsPublicKey,
					},
				}}, nil)
				remoteRunnerFactory.Returns(remoteRunner, nil)
			})

			It("uses the specified port", func() {
				Expect(remoteRunnerFactory.CallCount()).To(Equal(1))
				host, username, privateKey, _, hostPublicKeyAlgorithm, logger := remoteRunnerFactory.ArgsForCall(0)
				Expect(host).To(Equal("10.0.0.0:3457"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))
			})
		})

		Context("finds instances for the deployment, having multiple instances in an instance group", func() {
			var instance0Jobs, instance1Jobs orchestrator.Jobs
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{
					{
						JobName: "job1",
						ID:      "id1",
						Index:   newIndex(1),
					},
					{
						JobName: "job1",
						ID:      "id2",
						Index:   newIndex(0),
					},
				}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:      "username",
						Host:          "10.0.0.1",
						IndexOrID:     "id1",
						HostPublicKey: hostsPublicKey,
					},
					{
						Username:      "username",
						Host:          "10.0.0.2",
						IndexOrID:     "id2",
						HostPublicKey: hostsPublicKey,
					},
				}}, nil)
				remoteRunnerFactory.Returns(remoteRunner, nil)

				instance0Jobs = []orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						"",
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/bbr/backup"},
						instance.Metadata{},
						false,
						false,
					),
				}
				instance1Jobs = []orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						"",
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/bbr/backup"},
						instance.Metadata{},
						false,
						false,
					),
				}
				fakeJobFinder.FindJobsStub = func(instanceIdentifier instance.InstanceIdentifier, remoteRunner ssh.RemoteRunner, manifestQuerier instance.ManifestQuerier) (orchestrator.Jobs, error) {
					if instanceIdentifier.InstanceId == "id1" {
						return instance0Jobs, nil
					} else {
						return instance1Jobs, nil
					}
				}

				manifestQuerierCreator.Returns(manifestQuerier, nil)
			})

			It("collects the instances", func() {
				Expect(actualInstances).To(Equal([]orchestrator.Instance{
					bosh.NewBoshDeployedInstance(
						"job1",
						"1",
						"id1",
						remoteRunner,
						boshDeployment,
						false,
						boshLogger,
						instance0Jobs,
					),
					bosh.NewBoshDeployedInstance(
						"job1",
						"0",
						"id2",
						remoteRunner,
						boshDeployment,
						false,
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

			It("generates a manifest querier with the creator", func() {
				Expect(manifestQuerierCreator.CallCount()).To(Equal(1))
			})

			It("sets up ssh for each group found", func() {
				Expect(boshDeployment.SetUpSSHCallCount()).To(Equal(1))

				slug, opts := boshDeployment.SetUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job1", "")))
				Expect(opts).To(Equal(stubbedSshOpts))
			})

			It("creates a remote runner for each host", func() {
				Expect(remoteRunnerFactory.CallCount()).To(Equal(2))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger := remoteRunnerFactory.ArgsForCall(0)
				Expect(host).To(Equal("10.0.0.1"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger = remoteRunnerFactory.ArgsForCall(1)
				Expect(host).To(Equal("10.0.0.2"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))
			})

			It("finds the jobs with the job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(2))
			})
		})

		Context("finds instances for the deployment, having multiple instances, including a windows vm, in an instance group", func() {
			var instance0Jobs orchestrator.Jobs

			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{
					{
						JobName: "job1",
						ID:      "linux1",
						Index:   newIndex(0),
					},
					{
						JobName: "job1",
						ID:      "windows2",
						Index:   newIndex(1),
					},
				}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:      "username",
						Host:          "10.0.0.1",
						IndexOrID:     "linux1",
						HostPublicKey: hostsPublicKey,
					},
					{
						Username:      "username",
						Host:          "10.0.0.2",
						IndexOrID:     "windows2",
						HostPublicKey: hostsPublicKey,
					},
				}}, nil)

				remoteRunner.IsWindowsReturnsOnCall(0, false, nil)
				remoteRunner.IsWindowsReturnsOnCall(1, true, nil)

				remoteRunnerFactory.Returns(remoteRunner, nil)

				instance0Jobs = []orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						"",
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/bbr/backup"},
						instance.Metadata{},
						false,
						false,
					),
				}

				fakeJobFinder.FindJobsStub = func(instanceIdentifier instance.InstanceIdentifier, remoteRunner ssh.RemoteRunner, manifestQuerier instance.ManifestQuerier) (orchestrator.Jobs, error) {
					if instanceIdentifier.InstanceId == "linux1" {
						return instance0Jobs, nil
					} else {
						return nil, errors.New("should not call FindJobs on non-Linux VMs")
					}
				}

				manifestQuerierCreator.Returns(manifestQuerier, nil)
			})

			It("collects the instances", func() {
				Expect(actualInstances).To(Equal([]orchestrator.Instance{
					bosh.NewBoshDeployedInstance(
						"job1",
						"0",
						"linux1",
						remoteRunner,
						boshDeployment,
						false,
						boshLogger,
						instance0Jobs,
					),
				}))
			})

			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("checks the os is linux", func() {
				Expect(remoteRunner.IsWindowsCallCount()).To(Equal(2))
			})

			It("only finds the jobs with the job finder on the linux instance", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(1))
			})
		})

		Context("finds instances for the deployment, having multiple instances in multiple instance groups", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{
					{
						JobName: "job1",
						ID:      "id1",
						Index:   newIndex(0),
					},
					{
						JobName: "job1",
						ID:      "id2",
						Index:   newIndex(1),
					},
					{
						JobName: "job2",
						ID:      "id3",
						Index:   newIndex(0),
					},
					{
						JobName: "job2",
						ID:      "id4",
						Index:   newIndex(1),
					},
				}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHStub = func(slug director.AllOrInstanceGroupOrInstanceSlug, sshOpts director.SSHOpts) (director.SSHResult, error) {
					if slug.Name() == "job1" {
						return director.SSHResult{Hosts: []director.Host{
							{
								Username:      "username",
								Host:          "10.0.0.1",
								IndexOrID:     "id1",
								HostPublicKey: hostsPublicKey,
							},
							{
								Username:      "username",
								Host:          "10.0.0.2",
								IndexOrID:     "id2",
								HostPublicKey: hostsPublicKey,
							},
						}}, nil
					} else {
						return director.SSHResult{Hosts: []director.Host{
							{
								Username:      "username",
								Host:          "10.0.0.3",
								IndexOrID:     "id3",
								HostPublicKey: hostsPublicKey,
							},
							{
								Username:      "username",
								Host:          "10.0.0.4",
								IndexOrID:     "id4",
								HostPublicKey: hostsPublicKey,
							},
						}}, nil
					}
				}
				remoteRunnerFactory.Returns(remoteRunner, nil)
				fakeJobFinder.FindJobsStub = func(instanceIdentifier instance.InstanceIdentifier,
					remoteRunner ssh.RemoteRunner, manifestQuerier instance.ManifestQuerier) (orchestrator.Jobs, error) {
					if instanceIdentifier.InstanceGroupName == "job2" {
						return []orchestrator.Job{
							instance.NewJob(
								remoteRunner,
								"",
								boshLogger,
								"",
								instance.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/bbr/backup"},
								instance.Metadata{},
								false,
								false,
							),
						}, nil
					}

					return []orchestrator.Job{}, nil
				}
				manifestQuerierCreator.Returns(manifestQuerier, nil)
			})

			It("collects the instances", func() {
				Expect(actualInstances).To(Equal([]orchestrator.Instance{
					bosh.NewBoshDeployedInstance(
						"job1",
						"0",
						"id1",
						remoteRunner,
						boshDeployment,
						false,
						boshLogger,
						[]orchestrator.Job{},
					),
					bosh.NewBoshDeployedInstance(
						"job2",
						"0",
						"id3",
						remoteRunner,
						boshDeployment,
						false,
						boshLogger,
						[]orchestrator.Job{
							instance.NewJob(
								remoteRunner,
								"",
								boshLogger,
								"",
								instance.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/bbr/backup"},
								instance.Metadata{},
								false,
								false,
							),
						},
					),
					bosh.NewBoshDeployedInstance(
						"job2",
						"1",
						"id4",
						remoteRunner,
						boshDeployment,
						false,
						boshLogger,
						[]orchestrator.Job{
							instance.NewJob(
								remoteRunner,
								"",
								boshLogger,
								"",
								instance.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/bbr/backup"},
								instance.Metadata{},
								false,
								false,
							),
						},
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

			It("generates a manifest querier with the finder", func() {
				Expect(manifestQuerierCreator.CallCount()).To(Equal(1))
			})

			It("sets up ssh for each group found", func() {
				Expect(boshDeployment.SetUpSSHCallCount()).To(Equal(2))

				slug, opts := boshDeployment.SetUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job1", "")))
				Expect(opts).To(Equal(stubbedSshOpts))

				slug, opts = boshDeployment.SetUpSSHArgsForCall(1)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job2", "")))
				Expect(opts).To(Equal(stubbedSshOpts))
			})

			It("creates a remote runner for each host that has scripts, and the first instance of each group that doesn't", func() {
				Expect(remoteRunnerFactory.CallCount()).To(Equal(3))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger := remoteRunnerFactory.ArgsForCall(0)
				Expect(host).To(Equal("10.0.0.1"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger = remoteRunnerFactory.ArgsForCall(1)
				Expect(host).To(Equal("10.0.0.3"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger = remoteRunnerFactory.ArgsForCall(2)
				Expect(host).To(Equal("10.0.0.4"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))
			})

			It("for each remote runner, it finds the jobs with the job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(3))

				actualInstanceIdentifier, actualRemoteRunner, actualManifestQuerier := fakeJobFinder.FindJobsArgsForCall(0)
				Expect(actualInstanceIdentifier).To(Equal(instance.InstanceIdentifier{InstanceGroupName: "job1", InstanceId: "id1"}))
				Expect(actualRemoteRunner).To(Equal(remoteRunner))
				Expect(actualManifestQuerier).To(Equal(manifestQuerier))

				actualInstanceIdentifier, actualRemoteRunner, actualManifestQuerier = fakeJobFinder.FindJobsArgsForCall(1)
				Expect(actualInstanceIdentifier).To(Equal(instance.InstanceIdentifier{InstanceGroupName: "job2", InstanceId: "id3"}))
				Expect(actualRemoteRunner).To(Equal(remoteRunner))
				Expect(actualManifestQuerier).To(Equal(manifestQuerier))

				actualInstanceIdentifier, actualRemoteRunner, actualManifestQuerier = fakeJobFinder.FindJobsArgsForCall(2)
				Expect(actualInstanceIdentifier).To(Equal(instance.InstanceIdentifier{InstanceGroupName: "job2", InstanceId: "id4"}))
				Expect(actualRemoteRunner).To(Equal(remoteRunner))
				Expect(actualManifestQuerier).To(Equal(manifestQuerier))
			})
		})

		Context("failures", func() {
			var expectedError = "er ma gerd"

			Context("fails to find the deployment", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(nil, errors.New(expectedError))
				})

				It("does fail", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("tries to fetch deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
			})

			Context("fails to find vms for a deployment", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns(nil, errors.New(expectedError))
				})

				It("does fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})
				It("tries to fetch vm infos", func() {
					Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
				})

				It("fetches deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
			})

			Context("fails when vm info does not have an index", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{{
						JobName: "job1",
						ID:      "jobID",
					}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
					boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
						{
							Username:      "username",
							Host:          "10.0.0.0",
							IndexOrID:     "jobID",
							HostPublicKey: hostsPublicKey,
						},
					}}, nil)
					remoteRunnerFactory.Returns(remoteRunner, nil)
				})
				It("does fail", func() {
					Expect(actualError).To(HaveOccurred())
					Expect(actualError.Error()).To(ContainSubstring("couldn't find instance index"))
					Expect(actualError.Error()).To(ContainSubstring("vmInfo index is nil"))
				})
			})

			Context("fails when vm info does not match the instances in the ssh results", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{{
						JobName: "job1",
						ID:      "jobID",
						Index:   newIndex(0),
					}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
					boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
						{
							Username:      "username",
							Host:          "10.0.0.0",
							IndexOrID:     "otherJobID",
							HostPublicKey: hostsPublicKey,
						},
					}}, nil)
					remoteRunnerFactory.Returns(remoteRunner, nil)
				})
				It("does fail", func() {
					Expect(actualError).To(HaveOccurred())
					Expect(actualError.Error()).To(ContainSubstring("couldn't find instance index"))
					Expect(actualError.Error()).To(ContainSubstring("vmInfo does not contain given vmID"))
				})
			})

			Context("fails to generate ssh opts", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)

					optsGenerator.Returns(director.SSHOpts{}, "", errors.New(expectedError))
				})
				It("does fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
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
					Expect(actualError).To(MatchError(ContainSubstring("invalid instance group name")))
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
					boshDeployment.SetUpSSHReturns(director.SSHResult{}, errors.New(expectedError))
				})

				It("does fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
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

			Context("fails creating a remote runner, to the vm", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{{
						JobName: "job1",
					}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
					boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
						{
							Username:      "username",
							Host:          "10.0.0.0",
							IndexOrID:     "index",
							HostPublicKey: hostsPublicKey,
						},
					}}, nil)
					remoteRunnerFactory.Returns(nil, errors.New(expectedError))
				})

				It("does fail", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("tries to connect to the vm", func() {
					Expect(remoteRunnerFactory.CallCount()).To(Equal(1))
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

				It("cleanup the ssh user from the instance", func() {
					Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				})
			})

			Context("succeeds creating remote runners for some vms, fails others", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{
						{
							JobName: "job1",
							ID:      "jobID",
							Index:   newIndex(0),
						},
						{
							JobName: "job2",
							ID:      "jobID2",
							Index:   newIndex(0),
						}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)

					boshDeployment.SetUpSSHStub = func(slug director.AllOrInstanceGroupOrInstanceSlug, opts director.SSHOpts) (director.SSHResult, error) {
						if slug.Name() == "job1" {
							return director.SSHResult{Hosts: []director.Host{
								{
									Username:      "username",
									Host:          "10.0.0.0",
									IndexOrID:     "jobID",
									HostPublicKey: hostsPublicKey,
								},
							}}, nil
						} else {
							return director.SSHResult{}, errors.New(expectedError)
						}
					}
					remoteRunnerFactory.Returns(remoteRunner, nil)
				})

				It("fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("cleans up the successful SSH connection", func() {
					Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				})
			})

			Context("succeeds creating remote runners but fails to create instance group slug", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{
						{
							JobName: "job1",
							ID:      "jobID",
							Index:   newIndex(0),
						},
						{
							JobName: "job2/a/a/a",
							ID:      "jobID2",
							Index:   newIndex(0),
						}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)

					boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
						{
							Username:      "username",
							Host:          "10.0.0.0",
							IndexOrID:     "jobID",
							HostPublicKey: hostsPublicKey,
						},
					}}, nil)

					remoteRunnerFactory.Returns(remoteRunner, nil)
				})

				It("fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring("invalid instance group name")))
				})

				It("cleans up the successful SSH connection", func() {
					Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				})
			})

			Context("succeeds creating some remote runners but remote runner factory fails for a later connection", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{
						{
							JobName: "job1",
							ID:      "jobID",
							Index:   newIndex(0),
						},
						{
							JobName: "job2",
							ID:      "jobID2",
							Index:   newIndex(0),
						}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)

					boshDeployment.SetUpSSHStub = func(slug director.AllOrInstanceGroupOrInstanceSlug, opts director.SSHOpts) (director.SSHResult, error) {
						return director.SSHResult{Hosts: []director.Host{
							{
								Username:      "username",
								Host:          "10.0.0.0_" + slug.Name(),
								IndexOrID:     "jobID",
								HostPublicKey: hostsPublicKey,
							},
						}}, nil
					}

					remoteRunnerFactory.Stub = func(host, user, privateKey string, publicKeyCallback gossh.HostKeyCallback, publicKeyAlgorithm []string, logger ssh.Logger) (ssh.RemoteRunner, error) {
						if host == "10.0.0.0_job1" {
							return remoteRunner, nil
						}
						return nil, errors.New(expectedError)
					}
				})

				It("fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("cleans up the successful SSH connection", func() {
					Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(2))
				})
			})

			Context("fails when os checker returns an error", func() {
				var expectedErrorMessage = "failed to check os"

				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{
						{
							JobName: "job1",
						},
					}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)

					boshDeployment.SetUpSSHStub = func(slug director.AllOrInstanceGroupOrInstanceSlug, opts director.SSHOpts) (director.SSHResult, error) {
						return director.SSHResult{Hosts: []director.Host{
							{
								Username:      "username",
								Host:          "10.0.0.0_" + slug.Name(),
								IndexOrID:     "index",
								HostPublicKey: hostsPublicKey,
							},
						}}, nil
					}

					remoteRunner.IsWindowsReturns(false, errors.New(expectedErrorMessage))

					remoteRunnerFactory.Returns(remoteRunner, nil)
				})

				It("fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedErrorMessage)))
				})

				It("cleans up the successful SSH connection", func() {
					Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				})
			})
		})
	})

	Describe("GetManifest", func() {
		var actualManifest string
		var actualError error

		JustBeforeEach(func() {
			actualManifest, actualError = b.GetManifest(deploymentName)
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
				var findDeploymentError = "no deployment here"
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(nil, errors.New(findDeploymentError))
				})
				It("returns an error", func() {
					Expect(actualError).To(MatchError(ContainSubstring(findDeploymentError)))
				})
			})
			Context("to download manifest", func() {
				var downloadManifestError = errors.New("I refuse to download this manifest")
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.ManifestReturns("", downloadManifestError)
				})
				It("returns an error", func() {
					Expect(actualError).To(MatchError(downloadManifestError))
				})
			})
		})
	})
})

func newIndex(index int) *int {
	return &index
}
