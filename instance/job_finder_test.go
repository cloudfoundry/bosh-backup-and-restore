package instance_test

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/pivotal-cf/pcf-backup-and-restore/instance"
	"github.com/pivotal-cf/pcf-backup-and-restore/instance/fakes"

	"fmt"

	"bytes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"log"
)

var _ = Describe("JobFinderFromScripts", func() {
	var logStream *bytes.Buffer
	var jobFinder *JobFinderFromScripts
	var sshConnection *fakes.FakeSSHConnection
	var jobs Jobs
	var jobsError error

	Describe("FindJobs", func() {
		BeforeEach(func() {
			logStream = bytes.NewBufferString("")

			combinedLog := log.New(io.MultiWriter(GinkgoWriter, logStream), "[instance-test] ", log.Lshortfile)

			sshConnection = new(fakes.FakeSSHConnection)
			jobFinder = NewJobFinder(boshlog.New(boshlog.LevelDebug, combinedLog, combinedLog))
		})
		JustBeforeEach(func() {
			jobs, jobsError = jobFinder.FindJobs("identifier", sshConnection)
		})

		Context("has no job metadata scripts", func() {
			Context("Finds jobs based on scripts", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("/var/vcap/jobs/consul_agent/bin/b-backup\n"+
						"/var/vcap/jobs/consul_agent/bin/b-restore"), nil, 0, nil)
				})

				It("succeeds", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				It("finds the scripts", func() {
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/* -type f"))
				})

				It("returns a list of jobs", func() {
					Expect(jobs).To(Equal(NewJobs(BackupAndRestoreScripts{
						"/var/vcap/jobs/consul_agent/bin/b-backup",
						"/var/vcap/jobs/consul_agent/bin/b-restore",
					}, map[string]Metadata{})))
				})

				It("logs the scripts found", func() {
					Expect(logStream.String()).To(ContainSubstring("identifier/consul_agent/b-backup"))
					Expect(logStream.String()).To(ContainSubstring("identifier/consul_agent/b-restore"))
				})
			})

			Context("Finds invalid jobs scripts", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("/var/vcap/jobs/consul_agent/bin/foobar"), nil, 0, nil)
				})

				It("succeeds", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				It("finds the scripts", func() {
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/* -type f"))
				})

				It("returns a list of jobs", func() {
					Expect(jobs).To(Equal(NewJobs(BackupAndRestoreScripts{}, map[string]Metadata{})))
				})
			})

			Context("there are no job scripts returned from the connection", func() {
				BeforeEach(func() {
					sshConnection.RunReturns(
						nil, []byte("find: `/var/vcap/jobs/*/bin/*': No such file or directory"), 1, nil,
					)
				})

				It("does not fail", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				It("returns an empty list", func() {
					Expect(jobs).To(Equal(NewJobs(BackupAndRestoreScripts{}, map[string]Metadata{})))
				})
			})

			Context("find fails on a unknown error", func() {
				BeforeEach(func() {
					sshConnection.RunReturns(
						nil, []byte("find: `unknown error"), 1, nil,
					)
				})

				It("runs the command once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))

				})
				It("does fail", func() {
					Expect(jobsError).To(HaveOccurred())
				})
			})

			Context("find fails ssh connection error", func() {
				expectedError := fmt.Errorf("no!")

				BeforeEach(func() {
					sshConnection.RunReturns(
						nil, nil, 0, expectedError,
					)
				})

				It("runs the command once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))

				})
				It("does fail", func() {
					Expect(jobsError).To(MatchError(expectedError))
				})
			})
		})

		Context("ssh connection returns a metadata script", func() {
			Context("metadata is valid", func() {
				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/b-metadata" {
							return []byte(`---
backup_name: consul_backup`), nil, 0, nil
						}
						return []byte("/var/vcap/jobs/consul_agent/bin/b-metadata"), nil, 0, nil
					}

				})
				It("succeeds", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				It("uses the ssh connection to get the metadata", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(2))
					Expect(sshConnection.RunArgsForCall(1)).To(Equal("/var/vcap/jobs/consul_agent/bin/b-metadata"))

				})

				It("returns a list of jobs with metadata", func() {
					Expect(jobs).To(Equal(NewJobs(BackupAndRestoreScripts{
						"/var/vcap/jobs/consul_agent/bin/b-metadata",
					}, map[string]Metadata{
						"consul_agent": {BackupName: "consul_backup"},
					})))
				})
			})

			Context("reading metadata failed with ssh error", func() {
				expectedError := fmt.Errorf("foo!")

				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/b-metadata" {
							return []byte(`---
backup_name: consul_backup`), nil, 0, expectedError
						}
						return []byte("/var/vcap/jobs/consul_agent/bin/b-metadata"), nil, 0, nil
					}
				})

				It("fails", func() {
					Expect(jobsError.Error()).To(ContainSubstring(expectedError.Error()))
				})

				It("uses the ssh connection to get the metadata", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(2))
					Expect(sshConnection.RunArgsForCall(1)).To(Equal("/var/vcap/jobs/consul_agent/bin/b-metadata"))
				})
			})

			Context("reading metadata exited with non 0 code", func() {
				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/b-metadata" {
							return []byte(`---
backup_name: consul_backup`), nil, 0, nil
						}
						return []byte("/var/vcap/jobs/consul_agent/bin/b-metadata"), nil, 1, nil
					}
				})

				It("fails", func() {
					Expect(jobsError).To(HaveOccurred())
				})

				It("uses the ssh connection to find the metadata", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})
			})

			Context("reading metadata returned invalid yaml", func() {
				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/b-metadata" {
							return []byte(`they are being really unfair to me`), nil, 0, nil
						}
						return []byte("/var/vcap/jobs/consul_agent/bin/b-metadata"), nil, 0, nil
					}
				})

				It("succeeds", func() {
					Expect(jobsError).To(HaveOccurred())
				})

				It("uses the ssh connection to get the metadata", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(2))
					Expect(sshConnection.RunArgsForCall(1)).To(Equal("/var/vcap/jobs/consul_agent/bin/b-metadata"))
				})
			})
		})
	})
})
