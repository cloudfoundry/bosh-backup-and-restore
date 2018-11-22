package bosh_test

import (
	"fmt"
	"log"

	"errors"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	sshfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
	"github.com/cloudfoundry/bosh-cli/director"
	boshfakes "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("BoshDeployedInstance", func() {
	var remoteRunner *sshfakes.FakeRemoteRunner
	var boshDeployment *boshfakes.FakeDeployment
	var boshLogger boshlog.Logger
	var stdout *gbytes.Buffer
	var jobName, jobIndex, jobID string
	var artifactDirCreated bool
	var backuperInstance orchestrator.Instance

	BeforeEach(func() {
		remoteRunner = new(sshfakes.FakeRemoteRunner)
		boshDeployment = new(boshfakes.FakeDeployment)
		jobName = "job-name"
		jobIndex = "job-index"
		jobID = "job-id"
		stdout = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile))
		artifactDirCreated = true
	})

	JustBeforeEach(func() {
		remoteRunner.ConnectedUsernameReturns("sshUsername")
		backuperInstance = bosh.NewBoshDeployedInstance(
			jobName,
			jobIndex,
			jobID,
			remoteRunner,
			boshDeployment,
			artifactDirCreated,
			boshLogger,
			[]orchestrator.Job{},
		)
	})

	Describe("Cleanup", func() {
		var actualError error
		var expectedError error

		JustBeforeEach(func() {
			actualError = backuperInstance.Cleanup()
		})

		Describe("cleans up successfully", func() {
			It("deletes the backup folder", func() {
				Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(1))
				dir := remoteRunner.RemoveDirectoryArgsForCall(0)
				Expect(dir).To(Equal("/var/vcap/store/bbr-backup"))
			})

			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug(jobName, jobID)))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})

		Context("when the backup artifact directory was not created this time", func() {
			BeforeEach(func() {
				artifactDirCreated = false
			})

			It("does not delete the existing artifact", func() {
				Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(0))
			})

			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug(jobName, jobID)))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})

		Describe("error removing the backup folder", func() {
			BeforeEach(func() {
				expectedError = fmt.Errorf("foo bar")
				remoteRunner.RemoveDirectoryReturns(expectedError)
			})

			It("tries to cleanup ssh connection", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
			})

			It("returns the error", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})

		Describe("error removing the backup folder and an error while running cleaning up the connection", func() {
			var expectedErrorWhileDeleting error
			var expectedErrorWhileCleaningUp error

			BeforeEach(func() {
				expectedErrorWhileDeleting = fmt.Errorf("error while cleaning up var/vcap/store/bbr-backup")
				expectedErrorWhileCleaningUp = fmt.Errorf("error while cleaning the ssh tunnel")
				remoteRunner.RemoveDirectoryReturns(expectedErrorWhileDeleting)
				boshDeployment.CleanUpSSHReturns(expectedErrorWhileCleaningUp)
			})

			It("tries delete the artifact", func() {
				Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(1))
			})

			It("tries to cleanup ssh connection", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
			})

			It("returns the aggregated error", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedErrorWhileDeleting.Error())))
				Expect(actualError).To(MatchError(ContainSubstring(expectedErrorWhileCleaningUp.Error())))
			})
		})

		Describe("error while running cleaning up the connection", func() {
			BeforeEach(func() {
				expectedError = errors.New("werk niet")
				boshDeployment.CleanUpSSHReturns(expectedError)
			})

			It("fails", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})
	})

	Describe("CleanupPrevious", func() {
		var actualError error
		var expectedError error

		JustBeforeEach(func() {
			actualError = backuperInstance.CleanupPrevious()
		})

		Describe("cleans up successfully", func() {
			It("deletes the backup folder", func() {
				Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(1))
				dir := remoteRunner.RemoveDirectoryArgsForCall(0)
				Expect(dir).To(Equal("/var/vcap/store/bbr-backup"))
			})

			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug(jobName, jobID)))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})

		Context("when the backup artifact directory was not created this time", func() {
			BeforeEach(func() {
				artifactDirCreated = false
			})

			It("does attempt to delete the existing artifact", func() {
				Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(1))
				dir := remoteRunner.RemoveDirectoryArgsForCall(0)
				Expect(dir).To(Equal("/var/vcap/store/bbr-backup"))
			})

			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug(jobName, jobID)))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})

		Describe("error removing the backup folder", func() {
			BeforeEach(func() {
				expectedError = fmt.Errorf("foo bar")
				remoteRunner.RemoveDirectoryReturns(expectedError)
			})

			It("tries to cleanup ssh connection", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
			})

			It("returns the error", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})

		Describe("error removing the backup folder and an error while running cleaning up the connection", func() {
			var expectedErrorWhileDeleting error
			var expectedErrorWhileCleaningUp error

			BeforeEach(func() {
				expectedErrorWhileDeleting = fmt.Errorf("error while cleaning up var/vcap/store/bbr-backup")
				expectedErrorWhileCleaningUp = fmt.Errorf("error while cleaning the ssh tunnel")
				remoteRunner.RemoveDirectoryReturns(expectedErrorWhileDeleting)
				boshDeployment.CleanUpSSHReturns(expectedErrorWhileCleaningUp)
			})

			It("tries delete the artifact", func() {
				Expect(remoteRunner.RemoveDirectoryCallCount()).To(Equal(1))
			})

			It("tries to cleanup ssh connection", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
			})

			It("returns the aggregated error", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedErrorWhileDeleting.Error())))
				Expect(actualError).To(MatchError(ContainSubstring(expectedErrorWhileCleaningUp.Error())))
			})
		})

		Describe("error while running cleaning up the connection", func() {
			BeforeEach(func() {
				expectedError = errors.New("werk niet")
				boshDeployment.CleanUpSSHReturns(expectedError)
			})

			It("fails", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})
	})
})
