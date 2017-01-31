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
	var backupAndRestoreScripts []bosh.Script
	var jobs bosh.Jobs
	var blobNames map[string]string

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
		backupAndRestoreScripts = []bosh.Script{}
		blobNames = map[string]string{}
	})

	JustBeforeEach(func() {
		jobs, _ = bosh.NewJobs(backupAndRestoreScripts, blobNames)
		sshConnection.UsernameReturns("sshUsername")
		instance = bosh.NewBoshInstance(jobName, jobIndex, jobID, sshConnection, boshDeployment, boshLogger, jobs)
	})

	Describe("IsBackupable", func() {
		var actualBackupable bool

		JustBeforeEach(func() {
			actualBackupable = instance.IsBackupable()
		})

		Describe("there are backup scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/dave/bin/p-backup",
				}
			})

			It("returns true", func() {
				Expect(actualBackupable).To(BeTrue())
			})
		})

		Describe("there are no backup scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns false", func() {
				Expect(actualBackupable).To(BeFalse())
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

	Describe("IsRestorable", func() {
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

	Describe("PreBackupLock", func() {
		var err error

		JustBeforeEach(func() {
			err = instance.PreBackupLock()
		})

		Context("when there is one pre-backup-lock script in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{"/var/vcap/jobs/bar/bin/p-pre-backup-lock"}
			})

			It("uses the ssh connection to run the pre-backup-lock script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal(
					"sudo /var/vcap/jobs/bar/bin/p-pre-backup-lock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/p-pre-backup-lock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple backup scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-pre-backup-lock",
					"/var/vcap/jobs/bar/bin/p-pre-backup-lock",
					"/var/vcap/jobs/baz/bin/p-pre-backup-lock",
				}
			})

			It("uses the ssh connection to run each of the pre-backup-lock scripts", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo /var/vcap/jobs/foo/bin/p-pre-backup-lock",
					"sudo /var/vcap/jobs/bar/bin/p-pre-backup-lock",
					"sudo /var/vcap/jobs/baz/bin/p-pre-backup-lock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/p-pre-backup-lock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/p-pre-backup-lock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/p-pre-backup-lock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is locking the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Locking %s/%s for backup",
					jobName,
					jobID,
				)))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are several scripts and one of them fails to run pre backup lock while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("you are fake news")

			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-pre-backup-lock",
					"/var/vcap/jobs/bar/bin/p-pre-backup-lock",
					"/var/vcap/jobs/baz/bin/p-pre-backup-lock",
				}
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, expectedError
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("returns an error including the failure for the failed script", func() {
				Expect(err.Error()).To(ContainSubstring(
					fmt.Sprintf("pre backup lock script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("logs the failures related to the failed script", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(
					fmt.Sprintf("pre backup lock script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("returns an error without a message related to the script which passed", func() {
				Expect(err.Error()).NotTo(ContainSubstring(
					fmt.Sprintf("pre backup lock script for job foo failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			It("returns an error including the error from running the command", func() {
				Expect(err.Error()).To(ContainSubstring(expectedError.Error()))
			})

			It("logs the error caused when running the command", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Error attempting to run pre backup lock script for job baz on %s/%s. Error: %s",
					jobName,
					jobID,
					expectedError.Error(),
				)))
			})

			It("doesn't log Done", func() {
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("Done."))
			})
		})

	})

	Describe("Backup", func() {
		var err error

		JustBeforeEach(func() {
			err = instance.Backup()
		})

		Context("when there are multiple backup scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-backup",
					"/var/vcap/jobs/bar/bin/p-backup",
					"/var/vcap/jobs/baz/bin/p-backup",
				}
			})

			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo mkdir -p /var/vcap/store/backup/foo && sudo ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ /var/vcap/jobs/foo/bin/p-backup",
					"sudo mkdir -p /var/vcap/store/backup/bar && sudo ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ /var/vcap/jobs/bar/bin/p-backup",
					"sudo mkdir -p /var/vcap/store/backup/baz && sudo ARTIFACT_DIRECTORY=/var/vcap/store/backup/baz/ /var/vcap/jobs/baz/bin/p-backup",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/p-backup`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/p-backup`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/p-backup`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is backing up the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Backing up %s/%s",
					jobName,
					jobID,
				)))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple backup scripts and one of them is named", func() {
			BeforeEach(func() {
				blobNames = map[string]string{
					"baz": "special-backup",
				}
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-backup",
					"/var/vcap/jobs/bar/bin/p-backup",
					"/var/vcap/jobs/baz/bin/p-backup",
				}
			})

			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo mkdir -p /var/vcap/store/backup/foo && sudo ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ /var/vcap/jobs/foo/bin/p-backup",
					"sudo mkdir -p /var/vcap/store/backup/bar && sudo ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ /var/vcap/jobs/bar/bin/p-backup",
					"sudo mkdir -p /var/vcap/store/backup/special-backup && sudo ARTIFACT_DIRECTORY=/var/vcap/store/backup/special-backup/ /var/vcap/jobs/baz/bin/p-backup",
				))
			})
		})

		Context("when there are multiple jobs with no backup scripts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-restore",
					"/var/vcap/jobs/bar/bin/p-restore",
				}
			})
			It("makes calls to the instance over the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(0))
			})
		})

		Context("when there are several scripts and one of them fails to run backup while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("you are fake news")

			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-backup",
					"/var/vcap/jobs/bar/bin/p-backup",
					"/var/vcap/jobs/baz/bin/p-backup",
				}
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, expectedError
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("returns an error including the failure for the failed script", func() {
				Expect(err.Error()).To(ContainSubstring(
					fmt.Sprintf("backup script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("logs the failures related to the failed script", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(
					fmt.Sprintf("backup script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("returns an error without a message related to the script which passed", func() {
				Expect(err.Error()).NotTo(ContainSubstring(
					fmt.Sprintf("backup script for job foo failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			It("returns an error including the error from running the command", func() {
				Expect(err.Error()).To(ContainSubstring(expectedError.Error()))
			})

			It("logs the error caused when running the command", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Error attempting to run backup script for job baz on %s/%s. Error: %s",
					jobName,
					jobID,
					expectedError.Error(),
				)))
			})

			It("doesn't log Done", func() {
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("Done."))
			})
		})
	})

	Describe("PostBackupUnlock", func() {
		var err error

		JustBeforeEach(func() {
			err = instance.PostBackupUnlock()
		})

		Context("when there are multiple post-backup-unlock scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-post-backup-unlock",
					"/var/vcap/jobs/bar/bin/p-post-backup-unlock",
					"/var/vcap/jobs/baz/bin/p-post-backup-unlock",
				}
			})

			It("uses the ssh connection to run each post-backup-unlock script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo /var/vcap/jobs/foo/bin/p-post-backup-unlock",
					"sudo /var/vcap/jobs/bar/bin/p-post-backup-unlock",
					"sudo /var/vcap/jobs/baz/bin/p-post-backup-unlock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/p-post-backup-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/p-post-backup-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/p-post-backup-unlock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is backing up the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Unlocking %s/%s",
					jobName,
					jobID,
				)))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are several scripts and one of them fails to run post-backup-unlock while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("you are fake news")

			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-post-backup-unlock",
					"/var/vcap/jobs/bar/bin/p-post-backup-unlock",
					"/var/vcap/jobs/baz/bin/p-post-backup-unlock",
				}
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, expectedError
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("returns an error including the failure for the failed script", func() {
				Expect(err.Error()).To(ContainSubstring(
					fmt.Sprintf("unlock script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("logs the failures related to the failed script", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(
					fmt.Sprintf("unlock script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("returns an error without a message related to the script which passed", func() {
				Expect(err.Error()).NotTo(ContainSubstring(
					fmt.Sprintf("unlock script for job foo failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			It("returns an error including the error from running the command", func() {
				Expect(err.Error()).To(ContainSubstring(expectedError.Error()))
			})

			It("logs the error caused when running the command", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Error attempting to run unlock script for job baz on %s/%s. Error: %s",
					jobName,
					jobID,
					expectedError.Error(),
				)))
			})

			It("doesn't log Done", func() {
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("Done."))
			})
		})
	})

	Describe("Restore", func() {
		var actualError error

		JustBeforeEach(func() {
			actualError = instance.Restore()
		})

		Context("when there are multiple restore scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-restore",
					"/var/vcap/jobs/bar/bin/p-restore",
					"/var/vcap/jobs/baz/bin/p-restore",
				}
			})

			It("uses the ssh connection to run each restore script providing the correct ARTIFACT_DIRECTORTY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ /var/vcap/jobs/foo/bin/p-restore",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ /var/vcap/jobs/bar/bin/p-restore",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/backup/baz/ /var/vcap/jobs/baz/bin/p-restore",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/p-restore`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/p-restore`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/p-restore`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is restoring the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring to %s/%s",
					jobName,
					jobID,
				)))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
		})

		Context("when there are several scripts and one of them fails to run restore while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("i saw a million and a half people")

			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/foo/bin/p-restore",
					"/var/vcap/jobs/bar/bin/p-restore",
					"/var/vcap/jobs/baz/bin/p-restore",
				}
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, expectedError
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("returns an error including the failure for the failed script", func() {
				Expect(actualError.Error()).To(ContainSubstring(
					fmt.Sprintf("restore script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("logs the failures related to the failed script", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(
					fmt.Sprintf("restore script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("returns an error without a message related to the script which passed", func() {
				Expect(actualError.Error()).NotTo(ContainSubstring(
					fmt.Sprintf("restore script for job foo failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout from the failing job", func() {
				Expect(actualError.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr from the failing job", func() {
				Expect(actualError.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			It("returns an error including the error from running the command", func() {
				Expect(actualError.Error()).To(ContainSubstring(expectedError.Error()))
			})

			It("logs the error caused when running the command", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Error attempting to run restore script for job baz on %s/%s. Error: %s",
					jobName,
					jobID,
					expectedError.Error(),
				)))
			})

			It("doesn't log Done", func() {
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("Done."))
			})
		})
	})

	Describe("StreamBackupToRemote", func() {
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

	Describe("Cleanup", func() {
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

			It("tries delete the blob", func() {
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

	Describe("BackupChecksum", func() {
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

	Describe("Name", func() {
		It("returns the instance name", func() {
			Expect(instance.Name()).To(Equal("job-name"))
		})
	})

	Describe("Index", func() {
		It("returns the instance Index", func() {
			Expect(instance.Index()).To(Equal("job-index"))
		})
	})

	Describe("BackupSize", func() {
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

	Describe("Blobs", func() {
		var actualBlobs []backuper.BackupBlob

		JustBeforeEach(func() {
			actualBlobs = instance.Blobs()
		})

		Context("Has no named blobs", func() {
			It("returns the default blob", func() {
				Expect(actualBlobs).To(Equal([]backuper.BackupBlob{bosh.NewDefaultBlob(instance, sshConnection, boshLogger)}))
			})
		})

		Context("has a named blob", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []bosh.Script{
					"/var/vcap/jobs/job-name/bin/p-backup",
				}
				blobNames = map[string]string{
					"job-name": "my-blob",
				}
			})

			It("returns the named blob and the default blob", func() {
				Expect(actualBlobs).To(Equal(
					[]backuper.BackupBlob{
						bosh.NewNamedBlob(instance, bosh.NewJob(backupAndRestoreScripts, "my-blob"), sshConnection, boshLogger),
						bosh.NewDefaultBlob(instance, sshConnection, boshLogger),
					},
				))
			})
		})
	})

})
