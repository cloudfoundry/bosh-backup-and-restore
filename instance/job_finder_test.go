package instance_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance/fakes"
	sshfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
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
		var remoteRunner *sshfakes.FakeRemoteRunner
		var releaseMapping *fakes.FakeReleaseMapping
		var jobs orchestrator.Jobs
		var jobsError error

		consulAgentReleaseName := "consul-agent-release"

		BeforeEach(func() {
			remoteRunner = new(sshfakes.FakeRemoteRunner)
			remoteRunner.FindFilesReturns([]string{
				"/var/vcap/jobs/consul_agent/bin/bbr/backup",
				"/var/vcap/jobs/consul_agent/bin/bbr/restore",
				"/var/vcap/jobs/consul_agent/bin/bbr/pre-backup-lock",
				"/var/vcap/jobs/consul_agent/bin/bbr/pre-restore-lock",
				"/var/vcap/jobs/consul_agent/bin/bbr/post-backup-unlock",
				"/var/vcap/jobs/consul_agent/bin/bbr/post-restore-unlock"},
				nil)

			releaseMapping = new(fakes.FakeReleaseMapping)
			releaseMapping.FindReleaseNameReturns(consulAgentReleaseName, nil)
		})

		JustBeforeEach(func() {
			jobs, jobsError = jobFinder.FindJobs(instanceIdentifier, remoteRunner, releaseMapping)
		})

		It("finds the jobs", func() {
			By("finding the scripts", func() {
				Expect(remoteRunner.FindFilesArgsForCall(0)).To(Equal("/var/vcap/jobs/*/bin/bbr/*"))
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
					NewJob(remoteRunner, "identifier/0", logger, consulAgentReleaseName, BackupAndRestoreScripts{
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
				remoteRunner.FindFilesReturns([]string{"/var/vcap/jobs/consul_agent/bin/foobar"}, nil)
			})

			It("ignores them", func() {
				By("finding the scripts", func() {
					Expect(remoteRunner.FindFilesArgsForCall(0)).To(Equal("/var/vcap/jobs/*/bin/bbr/*"))
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
				remoteRunner.FindFilesReturns([]string{}, nil)
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

		Context("when find fails fails due to an error", func() {
			BeforeEach(func() {
				remoteRunner.FindFilesReturns(nil, fmt.Errorf("no! something bad has happened"))
			})

			It("fails", func() {
				By("calling find files", func() {
					Expect(remoteRunner.FindFilesCallCount()).To(Equal(1))
				})

				By("returning an error", func() {
					Expect(jobsError).To(MatchError(SatisfyAll(
						ContainSubstring("finding scripts failed on identifier/0"),
						ContainSubstring("no! something bad has happened"),
					)))
				})
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
					remoteRunner.FindFilesReturns([]string{"/var/vcap/jobs/consul_agent/bin/bbr/metadata"}, nil)
					remoteRunner.RunScriptReturns(`---
backup_name: consul_backup`, nil)
				})

				It("attaches the metadata to the corresponding jobs", func() {
					By("executing the metadata scripts", func() {
						Expect(remoteRunner.RunScriptArgsForCall(0)).To(Equal("/var/vcap/jobs/consul_agent/bin/bbr/metadata"))
					})

					By("adding the metadata to the returned jobs", func() {
						Expect(jobs).To(ConsistOf(
							NewJob(
								remoteRunner,
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
					remoteRunner.FindFilesReturns(nil, fmt.Errorf("ERROR"))
				})

				It("fails", func() {
					By("returning an error", func() {
						Expect(jobsError).To(MatchError(SatisfyAll(
							ContainSubstring("finding scripts failed on identifier/0"),
							ContainSubstring("ERROR"),
						)))
					})

					By("not trying to invoke the metadata scripts", func() {
						Expect(remoteRunner.RunScriptCallCount()).To(Equal(0))
					})
				})
			})

			Context("when executing a metadata script fails", func() {
				BeforeEach(func() {
					remoteRunner.FindFilesReturns([]string{"/var/vcap/jobs/consul_agent/bin/bbr/metadata"}, nil)
					remoteRunner.RunScriptReturns("", fmt.Errorf("blah blah blah foo"))
				})

				It("printing the location of the error, and the original error message", func() {
					Expect(jobsError).To(MatchError(ContainSubstring(
						"An error occurred while running metadata script for job consul_agent on identifier/0: blah blah blah foo",
					)))
				})
			})

			Context("when a metadata script returns invalid metadata YAML", func() {
				BeforeEach(func() {
					remoteRunner.FindFilesReturns([]string{"/var/vcap/jobs/consul_agent/bin/bbr/metadata"}, nil)
					remoteRunner.RunScriptReturns(`this metadata is missing all the keys`, nil)
				})

				It("prints the location of the error", func() {
					Expect(jobsError).To(MatchError(ContainSubstring(
						"Parsing metadata from job consul_agent on identifier/0 failed",
					)))
				})
			})
		})
	})
})
