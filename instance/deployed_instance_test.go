package instance_test

import (
	"errors"
	"fmt"
	"log"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh/fakes"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("DeployedInstance", func() {
	var sshConnection *fakes.FakeSSHConnection
	var boshLogger boshlog.Logger
	var stdout, stderr *gbytes.Buffer
	var jobName, jobIndex, jobID, expectedStdout, expectedStderr string
	var backupAndRestoreScripts []instance.Script
	var jobs instance.Jobs
	var blobMetadata map[string]instance.Metadata

	var backuperInstance *instance.DeployedInstance
	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		jobName = "job-name"
		jobIndex = "job-index"
		jobID = "job-id"
		expectedStdout = "i'm a stdout"
		expectedStderr = "i'm a stderr"
		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile), log.New(stderr, "[bosh-package] ", log.Lshortfile))
		backupAndRestoreScripts = []instance.Script{}
		blobMetadata = map[string]instance.Metadata{}
	})

	JustBeforeEach(func() {
		jobs = instance.NewJobs(backupAndRestoreScripts, blobMetadata)
		sshConnection.UsernameReturns("sshUsername")
		backuperInstance = instance.NewDeployedInstance(jobIndex, jobName, jobID, sshConnection, boshLogger, jobs)
	})

	Describe("IsBackupable", func() {
		var actualBackupable bool

		JustBeforeEach(func() {
			actualBackupable = backuperInstance.IsBackupable()
		})

		Describe("there are backup scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/bbr/backup",
				}
			})

			It("returns true", func() {
				Expect(actualBackupable).To(BeTrue())
			})
		})

		Describe("there are no backup scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
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

		JustBeforeEach(func() {
			actualLockable = backuperInstance.IsPreBackupLockable()
		})

		Describe("there are pre-backup-lock scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/bbr/pre-backup-lock",
				}
			})

			It("returns true", func() {
				Expect(actualLockable).To(BeTrue())
			})
		})

		Describe("there are no pre-backup-lock scripts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns false", func() {
				Expect(actualLockable).To(BeFalse())
			})
		})
	})

	Describe("IsPostBackupUnlockable", func() {
		var actualUnlockable bool

		JustBeforeEach(func() {
			actualUnlockable = backuperInstance.IsPostBackupUnlockable()
		})

		Context("there are post-backup-unlock scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/bbr/post-backup-unlock",
				}
			})

			It("returns true", func() {
				Expect(actualUnlockable).To(BeTrue())
			})
		})

		Context("there are no post-backup-unlock scripts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns false", func() {
				Expect(actualUnlockable).To(BeFalse())
			})
		})
	})

	Describe("IsRestorable", func() {
		var actualRestorable bool

		JustBeforeEach(func() {
			actualRestorable = backuperInstance.IsRestorable()
		})

		Describe("there are restore scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/bbr/restore",
				}
			})

			It("returns true", func() {
				Expect(actualRestorable).To(BeTrue())
			})
		})

		Describe("there are no restore scripts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns false", func() {
				Expect(actualRestorable).To(BeFalse())
			})
		})
	})

	Describe("CustomBackupBlobNames", func() {
		Context("when the instance has custom blob names defined", func() {
			BeforeEach(func() {
				blobMetadata = map[string]instance.Metadata{
					"dave": {BackupName: "foo"},
				}
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns a list of the instance's custom blob names", func() {
				Expect(backuperInstance.CustomBackupBlobNames()).To(ConsistOf("foo"))
			})
		})

	})

	Describe("CustomRestoreBlobNames", func() {
		Context("when the instance has custom restore blob names defined", func() {
			BeforeEach(func() {
				blobMetadata = map[string]instance.Metadata{
					"dave": {RestoreName: "foo"},
				}
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns a list of the instance's custom restore blob names", func() {
				Expect(backuperInstance.CustomRestoreBlobNames()).To(ConsistOf("foo"))
			})
		})

	})

	Describe("PreBackupLock", func() {
		var err error

		JustBeforeEach(func() {
			err = backuperInstance.PreBackupLock()
		})

		Context("when there is one pre-backup-lock script in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{"/var/vcap/jobs/bar/bin/bbr/pre-backup-lock"}
			})

			It("uses the ssh connection to run the pre-backup-lock script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal(
					"sudo /var/vcap/jobs/bar/bin/bbr/pre-backup-lock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/pre-backup-lock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs the job being locked", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Locking bar on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Done")))
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple backup scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
					"/var/vcap/jobs/bar/bin/bbr/pre-backup-lock",
					"/var/vcap/jobs/baz/bin/bbr/pre-backup-lock",
				}
			})

			It("uses the ssh connection to run each of the pre-backup-lock scripts", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo /var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
					"sudo /var/vcap/jobs/bar/bin/bbr/pre-backup-lock",
					"sudo /var/vcap/jobs/baz/bin/bbr/pre-backup-lock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/pre-backup-lock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/pre-backup-lock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/pre-backup-lock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is locking the job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Locking foo on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Done")))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Locking bar on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Done")))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Locking baz on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Done")))
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
			expectedError := errors.New("Errororororor")

			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
					"/var/vcap/jobs/bar/bin/bbr/pre-backup-lock",
					"/var/vcap/jobs/baz/bin/bbr/pre-backup-lock",
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
		})

	})

	Describe("Backup", func() {
		var err error

		JustBeforeEach(func() {
			err = backuperInstance.Backup()
		})

		Context("when there are multiple backup scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/bar/bin/bbr/backup",
					"/var/vcap/jobs/baz/bin/bbr/backup",
				}
			})

			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct ARTIFACT_DIRECTORY and BBR_ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo mkdir -p /var/vcap/store/backup/foo && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ /var/vcap/jobs/foo/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/backup/bar && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ /var/vcap/jobs/bar/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/backup/baz && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/baz/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/baz/ /var/vcap/jobs/baz/bin/bbr/backup",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/backup`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/backup`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/backup`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is backing up the job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up foo on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up bar on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up baz on %s/%s",
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
				blobMetadata = map[string]instance.Metadata{
					"baz": {BackupName: "special-backup"},
				}
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/bar/bin/bbr/backup",
					"/var/vcap/jobs/baz/bin/bbr/backup",
				}
			})

			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct BBR_ARTIFACT_DIRECTORY and ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo mkdir -p /var/vcap/store/backup/foo && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ /var/vcap/jobs/foo/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/backup/bar && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ /var/vcap/jobs/bar/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/backup/special-backup && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/special-backup/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/special-backup/ /var/vcap/jobs/baz/bin/bbr/backup",
				))
			})
		})

		Context("when there are multiple jobs with no backup scripts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/restore",
					"/var/vcap/jobs/bar/bin/bbr/restore",
				}
			})
			It("makes calls to the instance over the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(0))
			})
		})

		Context("when there are several scripts and one of them fails to run backup while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("I have a problem with your code")

			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/bar/bin/bbr/backup",
					"/var/vcap/jobs/baz/bin/bbr/backup",
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

		})
	})

	Describe("PostBackupUnlock", func() {
		var err error

		JustBeforeEach(func() {
			err = backuperInstance.PostBackupUnlock()
		})

		Context("when there are multiple post-backup-unlock scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
					"/var/vcap/jobs/bar/bin/bbr/post-backup-unlock",
					"/var/vcap/jobs/baz/bin/bbr/post-backup-unlock",
				}
			})

			It("uses the ssh connection to run each post-backup-unlock script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo /var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
					"sudo /var/vcap/jobs/bar/bin/bbr/post-backup-unlock",
					"sudo /var/vcap/jobs/baz/bin/bbr/post-backup-unlock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/post-backup-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/post-backup-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/post-backup-unlock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is backing up the job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking foo on %s/%s",
					jobName,
					jobID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking bar on %s/%s",
					jobName,
					jobID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking baz on %s/%s",
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
			expectedError := fmt.Errorf("I still have a problem with your code")

			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
					"/var/vcap/jobs/bar/bin/bbr/post-backup-unlock",
					"/var/vcap/jobs/baz/bin/bbr/post-backup-unlock",
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

		})
	})

	Describe("Restore", func() {
		var actualError error

		JustBeforeEach(func() {
			actualError = backuperInstance.Restore()
		})

		Context("when there are multiple restore scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/restore",
					"/var/vcap/jobs/bar/bin/bbr/restore",
					"/var/vcap/jobs/baz/bin/bbr/restore",
				}
			})

			It("uses the ssh connection to run each restore script providing the correct ARTIFACT_DIRECTORTY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ /var/vcap/jobs/foo/bin/bbr/restore",
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ /var/vcap/jobs/bar/bin/bbr/restore",
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/baz/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/baz/ /var/vcap/jobs/baz/bin/bbr/restore",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/restore`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/restore`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/restore`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is restoring a job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring foo on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring bar on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring baz on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))

			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple restore scripts and one of them is named", func() {
			BeforeEach(func() {
				blobMetadata = map[string]instance.Metadata{
					"baz": {RestoreName: "special-backup"},
				}
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/restore",
					"/var/vcap/jobs/bar/bin/bbr/restore",
					"/var/vcap/jobs/baz/bin/bbr/restore",
				}
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct BBR_ARTIFACT_DIRECTORY and ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/foo/ /var/vcap/jobs/foo/bin/bbr/restore",
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/bar/ /var/vcap/jobs/bar/bin/bbr/restore",
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/backup/special-backup/ ARTIFACT_DIRECTORY=/var/vcap/store/backup/special-backup/ /var/vcap/jobs/baz/bin/bbr/restore",
				))
			})
		})

		Context("when there are several scripts and one of them fails to run restore while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("foo bar baz error")

			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/restore",
					"/var/vcap/jobs/bar/bin/bbr/restore",
					"/var/vcap/jobs/baz/bin/bbr/restore",
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

		})
	})

	Describe("Name", func() {
		It("returns the instance name", func() {
			Expect(backuperInstance.Name()).To(Equal("job-name"))
		})
	})

	Describe("Index", func() {
		It("returns the instance Index", func() {
			Expect(backuperInstance.Index()).To(Equal("job-index"))
		})
	})

	Describe("BackupBlobs", func() {
		var backupBlobs []orchestrator.BackupBlob

		JustBeforeEach(func() {
			backupBlobs = backuperInstance.BlobsToBackup()
		})

		Context("Has no named backup blobs", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
				}
			})
			It("returns the default blob", func() {
				Expect(backupBlobs).To(Equal([]orchestrator.BackupBlob{instance.NewDefaultBlob(backuperInstance, sshConnection, boshLogger)}))
			})
		})

		Context("Has a named backup blob and a default blob", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/job-name/bin/bbr/backup",
				}
				blobMetadata = map[string]instance.Metadata{
					"job-name": {BackupName: "my-blob"},
				}
			})

			It("returns the named blob and the default blob", func() {
				Expect(backupBlobs).To(Equal(
					[]orchestrator.BackupBlob{
						instance.NewNamedBackupBlob(backuperInstance, instance.NewJob(
							backupAndRestoreScripts, instance.Metadata{BackupName: "my-blob"},
						), sshConnection, boshLogger),
						instance.NewDefaultBlob(backuperInstance, sshConnection, boshLogger),
					},
				))
			})

			It("returns the default blob the last", func() {
				Expect(backupBlobs[1]).To(Equal(instance.NewDefaultBlob(backuperInstance, sshConnection, boshLogger)))
			})

			It("returns the named blob first", func() {
				Expect(backupBlobs[0]).To(Equal(instance.NewNamedBackupBlob(
					backuperInstance, instance.NewJob(backupAndRestoreScripts, instance.Metadata{BackupName: "my-blob"}), sshConnection, boshLogger)))
			})
		})

		Context("Has only a named backup blob", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/job-name/bin/bbr/backup",
				}
				blobMetadata = map[string]instance.Metadata{
					"job-name": {BackupName: "my-blob"},
				}
			})

			It("returns the named blob and the default blob", func() {
				Expect(backupBlobs).To(Equal(
					[]orchestrator.BackupBlob{
						instance.NewNamedBackupBlob(backuperInstance, instance.NewJob(
							backupAndRestoreScripts, instance.Metadata{BackupName: "my-blob"},
						), sshConnection, boshLogger),
					},
				))
			})

		})
	})

	Describe("RestoreBlobs", func() {
		var restoreBlobs []orchestrator.BackupBlob

		JustBeforeEach(func() {
			restoreBlobs = backuperInstance.BlobsToRestore()
		})

		Context("Has no named restore blobs", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/job-name/bin/bbr/restore",
				}
			})
			It("returns the default blob", func() {
				Expect(restoreBlobs).To(Equal([]orchestrator.BackupBlob{instance.NewDefaultBlob(backuperInstance, sshConnection, boshLogger)}))
			})
		})

		Context("Has a named restore blob", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/job-name-2/bin/bbr/restore",
					"/var/vcap/jobs/job-name/bin/bbr/restore",
				}
				blobMetadata = map[string]instance.Metadata{
					"job-name": {RestoreName: "my-blob"},
				}
			})

			It("returns the named blob and the default blob", func() {
				Expect(restoreBlobs).To(Equal(
					[]orchestrator.BackupBlob{
						instance.NewDefaultBlob(backuperInstance, sshConnection, boshLogger),
						instance.NewNamedRestoreBlob(backuperInstance, instance.NewJob(
							backupAndRestoreScripts, instance.Metadata{RestoreName: "my-blob"},
						), sshConnection, boshLogger),
					},
				))
			})

			It("returns the default blob the first", func() {
				Expect(restoreBlobs[0]).To(Equal(instance.NewDefaultBlob(backuperInstance, sshConnection, boshLogger)))
			})

			It("returns the named blob last", func() {
				Expect(restoreBlobs[1]).To(Equal(instance.NewNamedRestoreBlob(
					backuperInstance, instance.NewJob(backupAndRestoreScripts, instance.Metadata{RestoreName: "my-blob"}), sshConnection, boshLogger)))
			})
		})

		Context("has only named restore blobs", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/job-name/bin/bbr/restore",
				}
				blobMetadata = map[string]instance.Metadata{
					"job-name": {RestoreName: "my-blob"},
				}
			})

			It("returns only the named blob", func() {
				Expect(restoreBlobs).To(Equal(
					[]orchestrator.BackupBlob{
						instance.NewNamedRestoreBlob(backuperInstance, instance.NewJob(
							backupAndRestoreScripts, instance.Metadata{RestoreName: "my-blob"},
						), sshConnection, boshLogger),
					},
				))
			})
		})
	})

})
