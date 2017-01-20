package bosh_test

import (
	"bytes"
	"fmt"
	"log"

	"errors"
	"github.com/cloudfoundry/bosh-cli/director"
	boshfakes "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh"
	"github.com/pivotal-cf/pcf-backup-and-restore/bosh/fakes"
	"strings"
)

var _ = Describe("Instance", func() {
	var sshConnection *fakes.FakeSSHConnection
	var boshDeployment *boshfakes.FakeDeployment
	var boshLogger boshlog.Logger
	var stdout, stderr *gbytes.Buffer
	var jobName, jobIndex, jobID, expectedStdout, expectedStderr string

	var instance backuper.Instance
	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		boshDeployment = new(boshfakes.FakeDeployment)
		jobName = "job-name"
		jobIndex = "job-index"
		jobID = "job-id"
		expectedStdout = "i'm a stdout"
		expectedStderr = "i'm a stderr"
		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile), log.New(stderr, "[bosh-package] ", log.Lshortfile))
	})

	JustBeforeEach(func() {
		sshConnection.UsernameReturns("sshUsername")
		instance = bosh.NewBoshInstance(jobName, jobIndex, jobID, sshConnection, boshDeployment, boshLogger,nil)
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
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-backup"))
			})

			Describe("when is backupable is called again", func() {
				var secondInvocationActualBackupable bool
				var secondInvocationActualError error
				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualBackupable, secondInvocationActualError = instance.IsBackupable()
				})

				It("only invokes the ssh connection once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})
				It("returns true", func() {
					Expect(secondInvocationActualBackupable).To(BeTrue())
				})
				It("succeeds", func() {
					Expect(secondInvocationActualError).NotTo(HaveOccurred())
				})
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
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-backup"))
			})

			Describe("when is backupable is called again", func() {
				var secondInvocationActualBackupable bool
				var secondInvocationActualError error
				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualBackupable, secondInvocationActualError = instance.IsBackupable()
				})

				It("only invokes the ssh connection once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})
				It("returns false", func() {
					Expect(secondInvocationActualBackupable).To(BeFalse())
				})
				It("succeeds", func() {
					Expect(secondInvocationActualError).NotTo(HaveOccurred())
				})
			})
		})

		Describe("error while running command", func() {
			var expectedError = fmt.Errorf("we need to build a wall")
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, expectedError)
			})
			It("fails", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-backup"))
			})

			Describe("when is backupable is called again", func() {
				var secondInvocationActualBackupable bool
				var secondInvocationActualError error
				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualBackupable, secondInvocationActualError = instance.IsBackupable()
				})

				It("invokes the ssh connection again", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(2))
				})

				It("fails", func() {
					Expect(secondInvocationActualError).To(HaveOccurred())
				})
			})
		})
	})

	Describe("IsPreBackupLockable", func() {
		var actualLockable bool
		var actualError error

		JustBeforeEach(func() {
			actualLockable, actualError = instance.IsPreBackupLockable()
		})

		Context("there are p-pre-backup-lock scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 0, nil)
			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("returns true", func() {
				Expect(actualLockable).To(BeTrue())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-pre-backup-lock"))
			})

			It("logs that we are checking for pre-backup-lock scripts", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Running check for pre-backup-lock scripts on %s/%s", jobName, jobID)))
			})

			It("logs stdout and stderr", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stdout: %s", expectedStdout)))
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			Describe("when is pre backup lockable is called again", func() {
				var secondInvocationActualLockable bool
				var secondInvocationActualError error
				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualLockable, secondInvocationActualError = instance.IsPreBackupLockable()
				})

				It("only invokes the ssh connection once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})

				It("returns true", func() {
					Expect(secondInvocationActualLockable).To(BeTrue())
				})

				It("succeeds", func() {
					Expect(secondInvocationActualError).NotTo(HaveOccurred())
				})
			})
		})

		Context("there are no p-pre-backup-lock scripts", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 1, nil)
			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("returns false", func() {
				Expect(actualLockable).To(BeFalse())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-pre-backup-lock"))
			})

			It("logs that we are checking for pre-backup-lock scripts", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Running check for pre-backup-lock scripts on %s/%s", jobName, jobID)))
			})

			It("logs stdout and stderr", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stdout: %s", expectedStdout)))
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})
		})

		Context("checking for p-pre-backup-lock scripts fails", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 0, fmt.Errorf("we have to deal with isis"))
			})

			It("fails", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("returns false", func() {
				Expect(actualLockable).To(BeFalse())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-pre-backup-lock"))
			})

			It("logs that we are checking for pre-backup-lock scripts", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Running check for pre-backup-lock scripts on %s/%s", jobName, jobID)))
			})

			It("logs stdout and stderr", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stdout: %s", expectedStdout)))
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})
		})
	})

	Describe("IsPostBackupUnlockable", func() {
		var actualUnlockable bool
		var actualError error

		JustBeforeEach(func() {
			actualUnlockable, actualError = instance.IsPostBackupUnlockable()
		})

		Context("there are p-post-backup-unlock scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 0, nil)
			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("returns true", func() {
				Expect(actualUnlockable).To(BeTrue())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-post-backup-unlock"))
			})

			It("logs that we are checking for post-backup-unlock scripts", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Running check for post-backup-unlock scripts on %s/%s", jobName, jobID)))
			})

			It("logs stdout and stderr", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stdout: %s", expectedStdout)))
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			Describe("when is post backup unlockable is called again", func() {
				var secondInvocationActualUnlockable bool
				var secondInvocationActualError error
				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualUnlockable, secondInvocationActualError = instance.IsPostBackupUnlockable()
				})

				It("only invokes the ssh connection once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})

				It("returns true", func() {
					Expect(secondInvocationActualUnlockable).To(BeTrue())
				})

				It("succeeds", func() {
					Expect(secondInvocationActualError).NotTo(HaveOccurred())
				})
			})
		})

		Context("there are no p-post-backup-unlock scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 1, nil)
			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("returns false", func() {
				Expect(actualUnlockable).To(BeFalse())
			})

			Describe("when is post backup unlockable is called again", func() {
				var secondInvocationActualUnlockable bool
				var secondInvocationActualError error

				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualUnlockable, secondInvocationActualError = instance.IsPostBackupUnlockable()
				})

				It("only invokes the ssh connection once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})

				It("returns false", func() {
					Expect(secondInvocationActualUnlockable).To(BeFalse())
				})

				It("succeeds", func() {
					Expect(secondInvocationActualError).NotTo(HaveOccurred())
				})
			})
		})

		Context("error while running command", func() {
			var expectedError = fmt.Errorf("we need to build a wall")

			BeforeEach(func() {
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 0, expectedError)
			})

			It("fails", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("logs the error to stderr", func() {
				Expect(stdout).To(
					gbytes.Say(
						fmt.Sprintf(
							"Error running check for post-backup-unlock scripts on instance %s/%s. Exit code 0, error: %s",
							jobName,
							jobID,
							expectedError,
						),
					),
				)
			})

			Describe("when is post backup backupable is called again", func() {
				var secondInvocationActualUnlockable bool
				var secondInvocationActualError error
				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualUnlockable, secondInvocationActualError = instance.IsPostBackupUnlockable()
				})

				It("invokes the ssh connection again", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(2))
				})

				It("fails", func() {
					Expect(secondInvocationActualError).To(HaveOccurred())
				})
			})
		})
	})

	Context("IsRestorable", func() {
		var actualRestorable bool
		var actualError error

		JustBeforeEach(func() {
			actualRestorable, actualError = instance.IsRestorable()
		})

		Describe("there are restore scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, nil)
			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("returns true", func() {
				Expect(actualRestorable).To(BeTrue())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/p-restore"))
			})

			Describe("when is restoreable is called again", func() {
				var secondInvocationActualRestorable bool
				var secondInvocationActualError error

				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualRestorable, secondInvocationActualError = instance.IsRestorable()
				})

				It("only invokes the ssh connection once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})

				It("returns true", func() {
					Expect(secondInvocationActualRestorable).To(BeTrue())
				})

				It("succeeds", func() {
					Expect(secondInvocationActualError).NotTo(HaveOccurred())
				})
			})

		})

		Describe("there are no restore scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 1, nil)
			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("returns false", func() {
				Expect(actualRestorable).To(BeFalse())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/p-restore"))
			})

			Describe("when is restoreable is called again", func() {
				var secondInvocationActualRestorable bool
				var secondInvocationActualError error

				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualRestorable, secondInvocationActualError = instance.IsRestorable()
				})

				It("only invokes the ssh connection once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})

				It("returns false", func() {
					Expect(secondInvocationActualRestorable).To(BeFalse())
				})

				It("succeeds", func() {
					Expect(secondInvocationActualError).NotTo(HaveOccurred())
				})
			})
		})

		Describe("error while running command", func() {
			var expectedError = fmt.Errorf("we need to build a wall")
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, expectedError)
			})
			It("fails", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("invokes the ssh connection, to find files", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("ls /var/vcap/jobs/*/bin/p-restore"))
			})

			Describe("when is restorable is called again", func() {
				var secondInvocationActualRestorable bool
				var secondInvocationActualError error
				JustBeforeEach(func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					secondInvocationActualRestorable, secondInvocationActualError = instance.IsRestorable()
				})

				It("invokes the ssh connection again", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(2))
				})

				It("fails", func() {
					Expect(secondInvocationActualError).To(HaveOccurred())
				})
			})
		})
	})

	Context("PreBackupLock", func() {
		var err error
		expectedError := fmt.Errorf("something went very wrong")

		JustBeforeEach(func() {
			err = instance.PreBackupLock()
		})

		It("succeeds", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("uses the ssh connection to find any pre-backup-lock scripts, and run them", func() {
			Expect(sshConnection.RunCallCount()).To(Equal(2))
			Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-pre-backup-lock"))
			Expect(sshConnection.RunArgsForCall(1)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-pre-backup-lock | xargs -IN sudo sh -c N"))
		})

		Describe("when there is an error with the ssh tunnel", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("some stdout"), []byte("some stderr"), 0, expectedError)
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("when the pre-backup-lock script returns an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"

			BeforeEach(func() {
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 1, nil)
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("prints an error message", func() {
				Expect(err.Error()).To(ContainSubstring(
					fmt.Sprintf("One or more pre-backup-lock scripts failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
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
				sshConnection.RunReturns([]byte("/var/vcap/foo/bar/backup\n/var/vcap/foo/baz/backup\n"), []byte("not relevant"), 0, nil)
			})

			It("uses the ssh connection to create the backup dir, and list + run all backup scripts as sudo", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(2))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-backup"))
				Expect(sshConnection.RunArgsForCall(1)).To(Equal("sudo mkdir -p /var/vcap/store/backup && ls /var/vcap/jobs/*/bin/p-backup | xargs -IN sudo sh -c N"))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/foo/bar/backup`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/foo/baz/backup`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("when there is an error backing up", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"

			BeforeEach(func() {
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "xargs") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("uses the ssh connection to create the backup dir and list + run all backup scripts as sudo", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(2))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-backup"))
				Expect(sshConnection.RunArgsForCall(1)).To(Equal("sudo mkdir -p /var/vcap/store/backup && ls /var/vcap/jobs/*/bin/p-backup | xargs -IN sudo sh -c N"))
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("prints an error message", func() {
				Expect(err.Error()).To(ContainSubstring(
					fmt.Sprintf("One or more backup scripts failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})
		})
	})

	Describe("PostBackupUnlock", func() {
		var err error

		JustBeforeEach(func() {
			err = instance.PostBackupUnlock()
		})

		Context("when there are post backup unlock scripts in the job directories", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 0, nil)
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("it uses the ssh connection to execute the post backup unlock scripts", func() {
				Expect(sshConnection.RunArgsForCall(0)).To(
					Equal("sudo ls /var/vcap/jobs/*/bin/p-post-backup-unlock | xargs -IN sudo sh -c N"),
				)
			})

			It("logs that we are running post backup unlock on the instance", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Running post backup unlock on %s/%s", jobName, jobID)))
				Expect(stdout).To(gbytes.Say("Done."))
			})

			It("logs stdout and stderr", func() {
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stdout: %s", expectedStdout)))
				Expect(stdout).To(gbytes.Say(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})
		})

		Context("when there is a post backup unlock script which fails", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 1, nil)
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("prints an error message", func() {
				Expect(err.Error()).To(ContainSubstring(
					fmt.Sprintf("One or more post-backup-unlock scripts failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})
		})

		Context("when there is an error executing the post backup unlock scripts", func() {
			var expectedError error

			BeforeEach(func() {
				expectedError = fmt.Errorf("you just picked a whole bunch of oopsie daisies")
				sshConnection.RunReturns([]byte(expectedStdout), []byte(expectedStderr), 0, expectedError)
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("returns the error from the SSH connection", func() {
				Expect(err).To(Equal(expectedError))
			})

			It("logs the error to stderr", func() {
				Expect(stderr).To(
					gbytes.Say(
						fmt.Sprintf(
							"Error running post backup unlock on instance %s/%s. Error: %s",
							jobName,
							jobID,
							expectedError,
						),
					),
				)
			})
		})
	})

	Context("Restore", func() {
		var actualError error

		JustBeforeEach(func() {
			actualError = instance.Restore()
		})

		Describe("runs restore successfully", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, nil)
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("lists, then runs, all restore scripts", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(2))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo ls /var/vcap/jobs/*/bin/p-restore"))
				Expect(sshConnection.RunArgsForCall(1)).To(Equal("ls /var/vcap/jobs/*/bin/p-restore | xargs -IN sudo sh -c N"))
			})
		})

		Describe("error while running command", func() {
			var expectedError = fmt.Errorf("we need to build a wall")
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, expectedError)
			})
			It("returns error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		Describe("restore scripts return an error", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("my fingers are long and beautiful"), 1, nil)
			})
			It("returns error", func() {
				Expect(actualError.Error()).To(ContainSubstring("Instance restore scripts returned %d. Error: %s", 1, "my fingers are long and beautiful"))
			})
		})
	})

	Context("StreamBackupToRemote", func() {
		var err error
		var reader = bytes.NewBufferString("dave")

		JustBeforeEach(func() {
			err = instance.StreamBackupToRemote(reader)
		})

		Describe("when successful", func() {
			It("uses the ssh connection to make the backup directory on the remote machine", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				command := sshConnection.RunArgsForCall(0)
				Expect(command).To(Equal("sudo mkdir -p /var/vcap/store/backup/"))
			})

			It("uses the ssh connection to stream files from the remote machine", func() {
				Expect(sshConnection.StreamStdinCallCount()).To(Equal(1))
				command, sentReader := sshConnection.StreamStdinArgsForCall(0)
				Expect(command).To(Equal("sudo sh -c 'tar -C /var/vcap/store/backup -zx'"))
				Expect(reader).To(Equal(sentReader))
			})

			It("does not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("when the remote side returns an error", func() {
			BeforeEach(func() {
				sshConnection.StreamStdinReturns([]byte("not relevant"), []byte("The beauty of me is that I’m very rich."), 1, nil)
			})

			It("fails and return the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("The beauty of me is that I’m very rich."))
			})
		})

		Describe("when there is an error running the stream", func() {
			BeforeEach(func() {
				sshConnection.StreamStdinReturns([]byte("not relevant"), []byte("not relevant"), 0, fmt.Errorf("My Twitter has become so powerful"))
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("My Twitter has become so powerful"))
			})
		})

		Describe("when creating the directory fails on the remote", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 1, nil)
			})

			It("fails and returns the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Creating backup directory on the remote returned 1"))
			})
		})

		Describe("when creating the directory fails because of a connection error", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, fmt.Errorf("These media people. The most dishonest people"))
			})

			It("fails and returns the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("These media people. The most dishonest people"))
			})
		})
	})

	Context("StreamBackupFromRemote", func() {
		var err error
		var writer = bytes.NewBufferString("dave")

		JustBeforeEach(func() {
			err = instance.StreamBackupFromRemote(writer)
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
			It("deletes the backup folder", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				cmd := sshConnection.RunArgsForCall(0)
				Expect(cmd).To(Equal("sudo rm -rf /var/vcap/store/backup"))
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
				sshConnection.RunReturns(nil, nil, 1, expectedError)
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
				expectedErrorWhileDeleting = fmt.Errorf("error while cleaning up var/vcap/store/backup")
				expectedErrorWhileCleaningUp = fmt.Errorf("error while cleaning the ssh tunnel")
				sshConnection.RunReturns(nil, nil, 1, expectedErrorWhileDeleting)
				boshDeployment.CleanUpSSHReturns(expectedErrorWhileCleaningUp)
			})

			It("tries delete the artifact", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
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
				expectedError = fmt.Errorf("werk niet")
				boshDeployment.CleanUpSSHReturns(expectedError)
			})
			It("fails", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})
	})

	Context("BackupChecksum", func() {
		var actualChecksum map[string]string
		var actualChecksumError error
		JustBeforeEach(func() {
			actualChecksum, actualChecksumError = instance.BackupChecksum()
		})
		Context("triggers find & shasum as root", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("not relevant"), nil, 0, nil)
			})
			It("generates the correct request", func() {
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("cd /var/vcap/store/backup; sudo sh -c 'find . -type f | xargs shasum'"))
			})
		})
		Context("can calculate checksum", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e  file1\n07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e  file2\nn87fc29fb3aacd99f7f7b81df9c43b13e71c56a1e file3/file4"), nil, 0, nil)
			})
			It("converts the checksum to a map", func() {
				Expect(actualChecksumError).NotTo(HaveOccurred())
				Expect(actualChecksum).To(Equal(map[string]string{
					"file1":       "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
					"file2":       "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
					"file3/file4": "n87fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
				}))
			})
		})
		Context("can calculate checksum, with trailing spaces", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e file1\n"), nil, 0, nil)
			})
			It("converts the checksum to a map", func() {
				Expect(actualChecksumError).NotTo(HaveOccurred())
				Expect(actualChecksum).To(Equal(map[string]string{
					"file1": "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
				}))
			})
		})
		Context("sha output is empty", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte(""), nil, 0, nil)
			})
			It("converts an empty map", func() {
				Expect(actualChecksumError).NotTo(HaveOccurred())
				Expect(actualChecksum).To(Equal(map[string]string{}))
			})
		})
		Context("sha for a empty directory", func() {
			BeforeEach(func() {
				sshConnection.RunReturns([]byte("da39a3ee5e6b4b0d3255bfef95601890afd80709  -"), nil, 0, nil)
			})
			It("reject '-' as a filename", func() {
				Expect(actualChecksumError).NotTo(HaveOccurred())
				Expect(actualChecksum).To(Equal(map[string]string{}))
			})
		})

		Context("fails to calculate checksum", func() {
			expectedErr := fmt.Errorf("some error")

			BeforeEach(func() {
				sshConnection.RunReturns(nil, nil, 0, expectedErr)
			})
			It("returns an error", func() {
				Expect(actualChecksumError).To(MatchError(expectedErr))
			})
		})
		Context("fails to execute the command", func() {
			BeforeEach(func() {
				sshConnection.RunReturns(nil, nil, 1, nil)
			})
			It("returns an error", func() {
				Expect(actualChecksumError).To(HaveOccurred())
			})
		})
	})

	Context("Name", func() {
		It("returns the instance name", func() {
			Expect(instance.Name()).To(Equal("job-name"))
		})
	})

	Context("Index", func() {
		It("returns the instance Index", func() {
			Expect(instance.Index()).To(Equal("job-index"))
		})
	})

	Context("BackupSize", func() {
		Context("when there is a backup", func() {
			var size string

			BeforeEach(func() {
				sshConnection.RunReturns([]byte("4.1G\n"), nil, 0, nil)
			})

			JustBeforeEach(func() {
				size, _ = instance.BackupSize()
			})

			It("returns the size of the backup according to the root user, as a string", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo du -sh /var/vcap/store/backup/ | cut -f1"))
				Expect(size).To(Equal("4.1G"))
			})
		})

		Context("when there is no backup directory", func() {
			var err error

			BeforeEach(func() {
				sshConnection.RunReturns(nil, nil, 1, nil) // simulating file not found
			})

			JustBeforeEach(func() {
				_, err = instance.BackupSize()
			})

			It("returns the size of the backup according to the root user, as a string", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo du -sh /var/vcap/store/backup/ | cut -f1"))
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when an error occurs", func() {
			var err error
			var actualError = errors.New("we will load it up with some bad dudes")

			BeforeEach(func() {
				sshConnection.RunReturns(nil, nil, 0, actualError)
			})

			JustBeforeEach(func() {
				_, err = instance.BackupSize()
			})

			It("returns the error", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(err).To(MatchError(actualError))
			})
		})

	})
})
