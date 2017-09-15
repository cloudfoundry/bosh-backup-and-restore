package instance_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	"fmt"

	"bytes"
	"io"
	"log"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("JobFinderFromScripts", func() {
	var logStream *bytes.Buffer
	var logger Logger
	var jobFinder *JobFinderFromScripts

	instanceIdentifier := InstanceIdentifier{InstanceGroupName: "identifier", InstanceId: "0"}

	BeforeEach(func() {
		logStream = bytes.NewBufferString("")
		combinedLog := log.New(io.MultiWriter(GinkgoWriter, logStream), "[instance-test] ", log.Lshortfile)
		logger = boshlog.New(boshlog.LevelDebug, combinedLog, combinedLog)

		jobFinder = NewJobFinder(logger)
	})

	Describe("FindJobs", func() {
		var sshConnection *fakes.FakeSSHConnection
		var releaseMapping *fakes.FakeReleaseMapping
		var jobs orchestrator.Jobs
		var jobsError error

		consulAgentReleaseName := "consul-agent-release"

		BeforeEach(func() {
			sshConnection = new(fakes.FakeSSHConnection)
			sshConnection.RunReturns([]byte(
				"/var/vcap/jobs/consul_agent/bin/bbr/backup\n"+
					"/var/vcap/jobs/consul_agent/bin/bbr/restore\n"+
					"/var/vcap/jobs/consul_agent/bin/bbr/pre-backup-lock\n"+
					"/var/vcap/jobs/consul_agent/bin/bbr/pre-restore-lock\n"+
					"/var/vcap/jobs/consul_agent/bin/bbr/post-backup-unlock\n"+
					"/var/vcap/jobs/consul_agent/bin/bbr/post-restore-unlock"),
				nil, 0, nil)

			releaseMapping = new(fakes.FakeReleaseMapping)
			releaseMapping.FindReleaseNameReturns(consulAgentReleaseName, nil)
		})

		JustBeforeEach(func() {
			jobs, jobsError = jobFinder.FindJobs(instanceIdentifier, sshConnection, releaseMapping)
		})

		It("finds the jobs", func() {
			By("finding the scripts", func() {
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/bbr/* -type f"))
			})

			By("logging the scripts found", func() {
				Expect(logStream.String()).To(ContainSubstring("identifier/0/consul_agent/backup"))
				Expect(logStream.String()).To(ContainSubstring("identifier/0/consul_agent/restore"))
				Expect(logStream.String()).To(ContainSubstring("identifier/0/consul_agent/pre-backup-lock"))
				Expect(logStream.String()).To(ContainSubstring("identifier/0/consul_agent/pre-restore-lock"))
				Expect(logStream.String()).To(ContainSubstring("identifier/0/consul_agent/post-backup-unlock"))
				Expect(logStream.String()).To(ContainSubstring("identifier/0/consul_agent/post-restore-unlock"))
			})

			By("calling `FindReleaseName` with the right arguments", func() {
				instanceGroupNameActual, jobNameActual := releaseMapping.FindReleaseNameArgsForCall(0)
				Expect(instanceGroupNameActual).To(Equal(instanceIdentifier.InstanceGroupName))
				Expect(jobNameActual).To(Equal("consul_agent"))
			})

			By("not returning an error", func() {
				Expect(jobsError).NotTo(HaveOccurred())
			})

			By("returning the list of jobs", func() {
				Expect(jobs).To(ConsistOf(
					NewJob(sshConnection, "identifier/0", logger, consulAgentReleaseName, BackupAndRestoreScripts{
						"/var/vcap/jobs/consul_agent/bin/bbr/backup",
						"/var/vcap/jobs/consul_agent/bin/bbr/restore",
						"/var/vcap/jobs/consul_agent/bin/bbr/post-backup-unlock",
						"/var/vcap/jobs/consul_agent/bin/bbr/post-restore-unlock",
						"/var/vcap/jobs/consul_agent/bin/bbr/pre-backup-lock",
						"/var/vcap/jobs/consul_agent/bin/bbr/pre-restore-lock",
					}, Metadata{})))
			})
		})

		Context("when invalid scripts are present", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("/var/vcap/jobs/consul_agent/bin/foobar"), nil, 0, nil)
			})

			It("ignores them", func() {
				By("finding the scripts", func() {
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/bbr/* -type f"))
				})

				By("not returning an error", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				By("returning an empty list of jobs", func() {
					Expect(jobs).To(BeEmpty())
				})
			})
		})

		Context("when scripts are missing", func() {
			BeforeEach(func() {
				sshConnection.RunReturns(
					nil, []byte("find: `/var/vcap/jobs/*/bin/bbr/*': No such file or directory"), 1, nil,
				)
			})

			It("does not return an error", func() {
				By("not returning an error", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				By("returning an empty list", func() {
					Expect(jobs).To(BeEmpty())
				})
			})
		})

		Context("when running `find` fails due to an unknown error", func() {
			BeforeEach(func() {
				sshConnection.RunReturns(
					nil, []byte("find: `unknown error"), 1, nil,
				)
			})

			It("fails", func() {
				By("running `find`", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})

				By("returning an error", func() {
					Expect(jobsError).To(HaveOccurred())
				})
			})
		})

		Context("when running `find` fails due to an SSH connection error", func() {
			expectedError := fmt.Errorf("no!")

			BeforeEach(func() {
				sshConnection.RunReturns(
					nil, nil, 0, expectedError,
				)
			})

			It("returns the SSH error", func() {
				Expect(jobsError).To(MatchError(expectedError))
			})
		})

		Context("when mapping jobs to releases fails", func() {
			var actualError = fmt.Errorf("release name mapping failure")

			BeforeEach(func() {
				releaseMapping.FindReleaseNameReturns("", actualError)
			})

			It("returns the error from the release mapper", func() {
				Expect(jobsError).To(MatchError(ContainSubstring(actualError.Error())))
			})
		})

		Context("when metadata scripts are present", func() {
			Context("when metadata is valid", func() {
				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/bbr/metadata" {
							return []byte(`---
backup_name: consul_backup`), nil, 0, nil
						}

						return []byte("/var/vcap/jobs/consul_agent/bin/bbr/metadata"), nil, 0, nil
					}
				})

				It("attaches the metadata to the corresponding jobs", func() {
					By("executing the metadata scripts", func() {
						Expect(sshConnection.RunArgsForCall(1)).To(Equal("/var/vcap/jobs/consul_agent/bin/bbr/metadata"))
					})

					By("adding the metadata to the returned jobs", func() {
						Expect(jobs).To(ConsistOf(
							NewJob(
								sshConnection,
								"identifier/0",
								logger,
								consulAgentReleaseName,
								BackupAndRestoreScripts{
									"/var/vcap/jobs/consul_agent/bin/bbr/metadata",
								}, Metadata{
									BackupName: "consul_backup",
								},
							),
						))
					})

					By("not returning an error", func() {
						Expect(jobsError).NotTo(HaveOccurred())
					})
				})
			})

			Context("when finding the scripts fails", func() {
				BeforeEach(func() {
					sshConnection.RunReturns(
						[]byte("/var/vcap/jobs/consul_agent/bin/bbr/metadata"), nil, 1, nil,
					)
				})

				It("fails", func() {
					By("returning an error", func() {
						Expect(jobsError).To(HaveOccurred())
					})

					By("not trying to invoke the metadata scripts", func() {
						Expect(sshConnection.RunCallCount()).To(Equal(1))
					})
				})
			})

			Context("when executing a metadata script fails", func() {
				expectedError := fmt.Errorf("foo!")

				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/bbr/metadata" {
							return []byte{}, []byte{}, 0, expectedError
						}

						return []byte("/var/vcap/jobs/consul_agent/bin/bbr/metadata"), nil, 0, nil
					}
				})

				It("returns the error from the SSH connection", func() {
					Expect(jobsError.Error()).To(ContainSubstring(expectedError.Error()))
				})
			})

			Context("when a metadata script returns a non-0 exis status", func() {
				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/bbr/metadata" {
							return []byte{}, []byte("STDERR"), 1, nil
						}

						return []byte("/var/vcap/jobs/consul_agent/bin/bbr/metadata"), nil, 0, nil
					}
				})

				It("fails", func() {
					By("returning an error", func() {
						Expect(jobsError).To(HaveOccurred())
					})

					By("using the contents of stderr as the error message", func() {
						Expect(jobsError).To(MatchError(ContainSubstring("STDERR")))
					})
				})
			})

			Context("when a metadata script returns invalid YAML", func() {
				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/bbr/metadata" {
							return []byte(`this is very disappointing`), nil, 0, nil
						}

						return []byte("/var/vcap/jobs/consul_agent/bin/bbr/metadata"), nil, 0, nil
					}
				})

				It("returns an error", func() {
					Expect(jobsError).To(MatchError(
						ContainSubstring("Reading job metadata for identifier/0 failed"),
					))
				})
			})
		})
	})
})
