package bosh_test

import (
	"bytes"
	"fmt"
	"log"

	"github.com/cloudfoundry/bosh-cli/director"
	boshfakes "github.com/cloudfoundry/bosh-cli/director/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh/fakes"
)

var _ = Describe("Instance", func() {
	var sshConnection *fakes.FakeSSHConnection
	var boshDeployment *boshfakes.FakeDeployment
	var boshLogger boshlog.Logger

	var instance backuper.Instance
	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		boshDeployment = new(boshfakes.FakeDeployment)
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(GinkgoWriter, "[bosh-package] ", log.Lshortfile), log.New(GinkgoWriter, "[bosh-package] ", log.Lshortfile))
	})

	JustBeforeEach(func() {
		sshConnection.UsernameReturns("sshUsername")
		instance = bosh.NewBoshInstance("job-name", "job-index", sshConnection, boshDeployment, boshLogger)
	})

	Context("IsBackupable", func() {
		var actualBackupable bool
		var actualError error

		JustBeforeEach(func() {
			actualBackupable, actualError = instance.IsBackupable()
		})

		Describe("there are backup scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, nil)
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
			It("returns true", func() {
				Expect(actualBackupable).To(BeTrue())
			})
			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/backup"))
			})
		})

		Describe("there are no backup scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 1, nil)
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
			It("returns false", func() {
				Expect(actualBackupable).To(BeFalse())
			})
			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/backup"))
			})
		})

		Describe("error while running command", func() {
			var expectedError = fmt.Errorf("we need to build a wall")
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, expectedError)
			})
			It("succeeds", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/backup"))
			})
		})
	})

	Context("Backup", func() {
		var err error

		JustBeforeEach(func() {
			err = instance.Backup()
		})
		Describe("when there are backup scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, nil)
			})
			It("uses the ssh connection to create the backup dir and run all backup scripts as sudo", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo mkdir -p /var/vcap/store/backup && ls /var/vcap/jobs/*/bin/backup | xargs -IN sudo sh -c N"))
			})
			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("when there is an error backing up", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 1, nil)
			})
			It("uses the ssh connection to create the backup dir and run all backup scripts as sudo", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo mkdir -p /var/vcap/store/backup && ls /var/vcap/jobs/*/bin/backup | xargs -IN sudo sh -c N"))
			})
			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("StreamBackupTo", func() {
		var err error
		var writer = bytes.NewBufferString("dave")

		JustBeforeEach(func() {
			err = instance.StreamBackupTo(writer)
		})

		Describe("when successful", func() {
			BeforeEach(func() {
				sshConnection.StreamReturns([]byte("not relevant"), 0, nil)
			})

			It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
				Expect(sshConnection.StreamCallCount()).To(Equal(1))

				cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
				Expect(cmd).To(Equal("sudo tar -C /var/vcap/store/backup -zc ."))
				Expect(returnedWriter).To(Equal(writer))
			})

			It("does not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("when there is an error tarring the backup", func() {
			BeforeEach(func() {
				sshConnection.StreamReturns([]byte("not relevant"), 1, nil)
			})

			It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
				Expect(sshConnection.StreamCallCount()).To(Equal(1))

				cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
				Expect(cmd).To(Equal("sudo tar -C /var/vcap/store/backup -zc ."))
				Expect(returnedWriter).To(Equal(writer))
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("when there is an SSH error", func() {
			var sshError error

			BeforeEach(func() {
				sshError = fmt.Errorf("I have the best SSH")
				sshConnection.StreamReturns([]byte("not relevant"), 0, sshError)
			})

			It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
				Expect(sshConnection.StreamCallCount()).To(Equal(1))

				cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
				Expect(cmd).To(Equal("sudo tar -C /var/vcap/store/backup -zc ."))
				Expect(returnedWriter).To(Equal(writer))
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(sshError))
			})
		})
	})

	Context("Cleanup", func() {
		var actualError error
		var expectedError error

		JustBeforeEach(func() {
			actualError = instance.Cleanup()
		})
		Describe("cleans up successfully", func() {
			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrPoolOrInstanceSlug("job-name", "job-index")))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})
		Describe("error while running delete", func() {
			BeforeEach(func() {
				expectedError = fmt.Errorf("werk niet")
				boshDeployment.CleanUpSSHReturns(expectedError)
			})
			It("fails", func() {
				Expect(actualError).To(MatchError(expectedError))
			})
		})
	})

	Context("Name", func() {
		It("returns the instance name", func() {
			Expect(instance.Name()).To(Equal("job-name"))
		})
	})

	Context("ID", func() {
		It("returns the instance ID", func() {
			Expect(instance.ID()).To(Equal("job-index"))
		})

	})
})
