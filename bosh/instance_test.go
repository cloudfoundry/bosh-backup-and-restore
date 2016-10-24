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

var _ = Describe("Instance", func() {
	var sshConnection *fakes.FakeSSHConnection
	var boshDeployment *boshfakes.FakeDeployment

	var instance backuper.Instance
	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		boshDeployment = new(boshfakes.FakeDeployment)
	})

	JustBeforeEach(func() {
		sshConnection.UsernameReturns("sshUsername")
		instance = bosh.NewBoshInstance("job-name", "job-index", sshConnection, boshDeployment)
	})

	Context("IsBackupable", func() {
		var actualBackupable bool
		var acutalError error

		JustBeforeEach(func() {
			actualBackupable, acutalError = instance.IsBackupable()
		})

		Describe("there are backup scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, nil)
			})
			It("succeeds", func() {
				Expect(acutalError).NotTo(HaveOccurred())
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
				Expect(acutalError).NotTo(HaveOccurred())
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
				Expect(acutalError).To(HaveOccurred())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/backup"))
			})
		})
	})

	Context("Cleanup", func() {
		var acutalError error
		var expectedError error

		JustBeforeEach(func() {
			acutalError = instance.Cleanup()
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
				Expect(acutalError).To(MatchError(expectedError))
			})
		})
	})
})
