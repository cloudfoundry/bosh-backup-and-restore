package instance_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("DeployedInstance", func() {
	var sshConnection *fakes.FakeSSHConnection
	var boshLogger boshlog.Logger
	var stdout, stderr *gbytes.Buffer
	var instanceGroupName, instanceIndex, instanceID, expectedStdout, expectedStderr string
	var jobs orchestrator.Jobs
	var remoteRunner instance.RemoteRunner

	var deployedInstance *instance.DeployedInstance
	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		instanceGroupName = "instance-group-name"
		instanceIndex = "instance-index"
		instanceID = "instance-id"
		expectedStdout = "i'm a stdout"
		expectedStderr = "i'm a stderr"
		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile), log.New(stderr, "[bosh-package] ", log.Lshortfile))
		remoteRunner = instance.NewRemoteRunner(sshConnection, boshLogger)
	})

	JustBeforeEach(func() {
		sshConnection.UsernameReturns("sshUsername")
		deployedInstance = instance.NewDeployedInstance(
			instanceIndex,
			instanceGroupName,
			instanceID,
			false,
			remoteRunner,
			boshLogger,
			jobs)
	})

	Describe("IsBackupable", func() {
		var actualBackupable bool

		JustBeforeEach(func() {
			actualBackupable = deployedInstance.IsBackupable()
		})

		Describe("there are backup scripts in the job directories", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/dave/bin/bbr/backup",
					}, instance.Metadata{}),
				})
			})

			It("returns true", func() {
				Expect(actualBackupable).To(BeTrue())
			})
		})

		Describe("there are no backup scripts in the job directories", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/dave/bin/foo",
					}, instance.Metadata{}),
				})
			})

			It("returns false", func() {
				Expect(actualBackupable).To(BeFalse())
			})
		})
	})

	Describe("ArtifactDirExists", func() {
		var sshExitCode int
		var sshError error

		var dirExists bool
		var dirError error

		JustBeforeEach(func() {
			sshConnection.RunReturns([]byte{}, []byte{}, sshExitCode, sshError)
			dirExists, dirError = deployedInstance.ArtifactDirExists()
		})

		BeforeEach(func() {
			sshExitCode = 1
		})

		Context("when artifact directory does not exist", func() {
			It("calls the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo stat /var/vcap/store/bbr-backup"))
			})

			It("returns false", func() {
				Expect(dirExists).To(BeFalse())
			})
		})

		Context("when artifact directory exists", func() {
			BeforeEach(func() {
				sshExitCode = 0
			})

			It("calls the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo stat /var/vcap/store/bbr-backup"))
			})

			It("returns true", func() {
				Expect(dirExists).To(BeTrue())
			})
		})

		Context("when ssh connection error occurs", func() {
			BeforeEach(func() {
				sshError = fmt.Errorf("argh!")
			})

			It("returns the error", func() {
				Expect(dirError).To(MatchError("argh!"))
			})
		})
	})

	Describe("IsRestorable", func() {
		var actualRestorable bool

		JustBeforeEach(func() {
			actualRestorable = deployedInstance.IsRestorable()
		})

		Describe("there are restore scripts in the job directories", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/dave/bin/bbr/restore",
					}, instance.Metadata{}),
				})
			})

			It("returns true", func() {
				Expect(actualRestorable).To(BeTrue())
			})
		})

		Describe("there are no restore scripts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/dave/bin/foo",
					}, instance.Metadata{}),
				})
			})

			It("returns false", func() {
				Expect(actualRestorable).To(BeFalse())
			})
		})
	})

	Describe("CustomBackupArtifactNames", func() {
		Context("when the instance has custom artifact names defined", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/dave/bin/foo",
					}, instance.Metadata{
						BackupName: "foo",
					}),
				})
			})

			It("returns a list of the instance's custom artifact names", func() {
				Expect(deployedInstance.CustomBackupArtifactNames()).To(ConsistOf("foo"))
			})
		})

	})

	Describe("CustomRestoreArtifactNames", func() {
		Context("when the instance has custom restore artifact names defined", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/dave/bin/foo",
					}, instance.Metadata{
						RestoreName: "foo",
					}),
				})
			})

			It("returns a list of the instance's custom restore artifact names", func() {
				Expect(deployedInstance.CustomRestoreArtifactNames()).To(ConsistOf("foo"))
			})
		})

	})

	Describe("Jobs", func() {
		BeforeEach(func() {
			jobs = orchestrator.Jobs([]orchestrator.Job{
				instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/dave/bin/foo",
				}, instance.Metadata{}),
			})
		})

		It("returns the instance's jobs", func() {
			Expect(deployedInstance.Jobs()).To(HaveLen(1))
			Expect(deployedInstance.Jobs()[0].Name()).To(Equal("dave"))
		})
	})

	Describe("Backup", func() {
		var err error

		JustBeforeEach(func() {
			err = deployedInstance.Backup()
		})

		Context("when there are multiple backup scripts in multiple job directories", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/backup",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/backup",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/backup",
					}, instance.Metadata{}),
				})
			})

			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct ARTIFACT_DIRECTORY and BBR_ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(6))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
					sshConnection.RunArgsForCall(3),
					sshConnection.RunArgsForCall(4),
					sshConnection.RunArgsForCall(5),
				}).To(ConsistOf(
					"sudo mkdir -p /var/vcap/store/bbr-backup/foo",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/bbr-backup/bar",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ /var/vcap/jobs/bar/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/bbr-backup/baz",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/baz/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/baz/ /var/vcap/jobs/baz/bin/bbr/backup",
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
					instanceGroupName,
					instanceID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up bar on %s/%s",
					instanceGroupName,
					instanceID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up baz on %s/%s",
					instanceGroupName,
					instanceID,
				)))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("marks the instance as having had its artifact directory created", func() {
				Expect(deployedInstance.ArtifactDirCreated()).To(BeTrue())
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple backup scripts and one of them is named", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/backup",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/backup",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/backup",
					}, instance.Metadata{BackupName: "special-backup"}),
				})
			})

			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct BBR_ARTIFACT_DIRECTORY and ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(6))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
					sshConnection.RunArgsForCall(3),
					sshConnection.RunArgsForCall(4),
					sshConnection.RunArgsForCall(5),
				}).To(ConsistOf(
					"sudo mkdir -p /var/vcap/store/bbr-backup/foo",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/bbr-backup/bar",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ /var/vcap/jobs/bar/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/bbr-backup/special-backup",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/special-backup/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/special-backup/ /var/vcap/jobs/baz/bin/bbr/backup",
				))
			})
		})

		Context("when there are multiple jobs with no backup scripts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/restore",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/restore",
					}, instance.Metadata{}),
				})
			})
			It("doesn't make calls to the instance over the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(0))
			})
		})

		Context("when there are several scripts and one of them fails to run backup while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("I have a problem with your code")

			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/backup",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/backup",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/backup",
					}, instance.Metadata{}),
				})
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
				By("including all relevant information", func() {
					Expect(err).To(MatchError(SatisfyAll(
						ContainSubstring(fmt.Sprintf("Error attempting to run backup for job bar on %s/%s.", instanceGroupName, instanceID)),
						ContainSubstring(expectedStderr),
						ContainSubstring(expectedError.Error()),
					)))
				})

				By("not including a message related to the script which passed", func() {
					Expect(err.Error()).NotTo(ContainSubstring(
						fmt.Sprintf("backup script for job foo failed on %s/%s", instanceGroupName, instanceID),
					))
				})
			})
		})
	})

	Describe("PostBackupUnlock", func() {
		var err error

		JustBeforeEach(func() {
			err = deployedInstance.PostBackupUnlock()
		})

		Context("when there are multiple post-backup-unlock scripts in multiple job directories", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/post-backup-unlock",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/post-backup-unlock",
					}, instance.Metadata{}),
				})
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
					instanceGroupName,
					instanceID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking bar on %s/%s",
					instanceGroupName,
					instanceID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking baz on %s/%s",
					instanceGroupName,
					instanceID,
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
			sshConnectionError := fmt.Errorf("I still have a problem with your code")

			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/post-backup-unlock",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/post-backup-unlock",
					}, instance.Metadata{}),
				})
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, sshConnectionError
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				By("including all relevant information", func() {
					Expect(err).To(MatchError(SatisfyAll(
						ContainSubstring(fmt.Sprintf("Error attempting to run post-backup-unlock for job baz on %s/%s", instanceGroupName, instanceID)),
						ContainSubstring(expectedStderr),
						ContainSubstring(sshConnectionError.Error()),
					)))
				})

				By("not including a message related to the script which passed", func() {
					Expect(err.Error()).NotTo(ContainSubstring(
						fmt.Sprintf("unlock script for job foo failed on %s/%s", instanceGroupName, instanceID),
					))
				})
			})
		})
	})

	Describe("PostRestoreUnlock", func() {
		var postRestoreUnlockError error

		JustBeforeEach(func() {
			postRestoreUnlockError = deployedInstance.PostRestoreUnlock()
		})

		Context("when there are multiple post-restore-unlock scripts in multiple job directories", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/post-restore-unlock",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/post-restore-unlock",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/post-restore-unlock",
					}, instance.Metadata{}),
				})
			})

			It("uses the ssh connection to run each post-restore-unlock script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo /var/vcap/jobs/foo/bin/bbr/post-restore-unlock",
					"sudo /var/vcap/jobs/bar/bin/bbr/post-restore-unlock",
					"sudo /var/vcap/jobs/baz/bin/bbr/post-restore-unlock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/post-restore-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/post-restore-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/post-restore-unlock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is unlocking the job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking foo on %s/%s",
					instanceGroupName,
					instanceID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking bar on %s/%s",
					instanceGroupName,
					instanceID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking baz on %s/%s",
					instanceGroupName,
					instanceID,
				)))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("succeeds", func() {
				Expect(postRestoreUnlockError).NotTo(HaveOccurred())
			})
		})

		Context("when there are several scripts and one of them fails to run post-restore-unlock while another one causes an error", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/post-restore-unlock",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/post-restore-unlock",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/post-restore-unlock",
					}, instance.Metadata{}),
				})
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte("stdout_bar"), []byte("stderr_bar"), 1, nil
					}

					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, fmt.Errorf("connection failed, script not run on baz")
					}

					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				By("including all relevant information", func() {
					Expect(postRestoreUnlockError).To(MatchError(SatisfyAll(
						ContainSubstring(fmt.Sprintf("Error attempting to run post-restore-unlock for job baz on %s/%s", instanceGroupName, instanceID)),
						ContainSubstring("stderr_bar"),
						ContainSubstring("connection failed, script not run on baz"),
					)))
				})
			})
		})

		Context("When there are some jobs without post-restore-unlock scripts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/job-has-unlock-script/bin/bbr/post-restore-unlock",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/job-only-has-backup/bin/bbr/backup",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/job-only-has-restore/bin/bbr/restore",
					}, instance.Metadata{}),
				})
			})

			It("Only invokes post-restore-unlock on those jobs which have that script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo /var/vcap/jobs/job-has-unlock-script/bin/bbr/post-restore-unlock"))
			})
		})
	})

	Describe("Restore", func() {
		var actualError error

		JustBeforeEach(func() {
			actualError = deployedInstance.Restore()
		})

		Context("when there are multiple restore scripts in multiple job directories", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/restore",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/restore",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/restore",
					}, instance.Metadata{}),
				})
			})

			It("uses the ssh connection to run each restore script providing the correct ARTIFACT_DIRECTORTY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/restore",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ /var/vcap/jobs/bar/bin/bbr/restore",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/baz/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/baz/ /var/vcap/jobs/baz/bin/bbr/restore",
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
					instanceGroupName,
					instanceID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring bar on %s/%s",
					instanceGroupName,
					instanceID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring baz on %s/%s",
					instanceGroupName,
					instanceID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))

			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple restore scripts and one of them is named", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/restore",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/restore",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/restore",
					}, instance.Metadata{RestoreName: "special-backup"}),
				})
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
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/restore",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ /var/vcap/jobs/bar/bin/bbr/restore",
					"sudo ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/special-backup/ BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/special-backup/ /var/vcap/jobs/baz/bin/bbr/restore",
				))
			})
		})

		Context("when there are several scripts and one of them fails to run restore while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("foo bar baz error")

			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/foo/bin/bbr/restore",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/bar/bin/bbr/restore",
					}, instance.Metadata{}),
					instance.NewJob(remoteRunner, instanceGroupName+"/"+instanceID, boshLogger, "", instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/baz/bin/bbr/restore",
					}, instance.Metadata{}),
				})
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
				By("including all relevant information", func() {
					Expect(actualError).To(MatchError(SatisfyAll(
						ContainSubstring(fmt.Sprintf("Error attempting to run restore for job baz on %s/%s", instanceGroupName, instanceID)),
						ContainSubstring(expectedStderr),
						ContainSubstring(expectedError.Error()),
					)))
				})

				By("not including a message related to the script which passed", func() {
					Expect(actualError.Error()).NotTo(ContainSubstring(
						fmt.Sprintf("restore script for job foo failed on %s/%s", instanceGroupName, instanceID),
					))
				})
			})
		})
	})

	Describe("Name", func() {
		It("returns the instance name", func() {
			Expect(deployedInstance.Name()).To(Equal("instance-group-name"))
		})
	})

	Describe("Index", func() {
		It("returns the instance Index", func() {
			Expect(deployedInstance.Index()).To(Equal("instance-index"))
		})
	})

	Describe("ArtifactsToBackup", func() {
		var backupArtifacts []orchestrator.BackupArtifact
		var instanceIdentifier instance.InstanceIdentifier

		var jobWithBackupScript1 = instance.NewJob(remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-backup-script-1/bin/bbr/backup"},
			instance.Metadata{})
		var jobWithBackupScript2 = instance.NewJob(remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-backup-script-2/bin/bbr/backup"},
			instance.Metadata{})
		var jobWithBackupScriptAndMetadata = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/job-with-backup-script-and-metadata/bin/bbr/backup",
			},
			instance.Metadata{
				BackupName: "my-artifact",
			},
		)
		var jobWithRestoreScript = instance.NewJob(remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-restore-script-1/bin/bbr/restore"},
			instance.Metadata{})
		var jobWithOnlyLockScript = instance.NewJob(remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-only-lock-script/bin/bbr/pre-backup-lock"},
			instance.Metadata{})

		BeforeEach(func() {
			instanceIdentifier = instance.InstanceIdentifier{InstanceGroupName: instanceGroupName, InstanceId: instanceID}
		})

		JustBeforeEach(func() {
			backupArtifacts = deployedInstance.ArtifactsToBackup()
		})

		Context("when the instance has no named backup artifacts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					jobWithBackupScript1,
					jobWithBackupScript2,
					jobWithRestoreScript,
				})
			})

			It("returns artifacts with default names", func() {
				Expect(backupArtifacts).To(ConsistOf(
					instance.NewBackupArtifact(jobWithBackupScript1, deployedInstance, instance.NewRemoteRunner(sshConnection, boshLogger), boshLogger),
					instance.NewBackupArtifact(jobWithBackupScript2, deployedInstance, instance.NewRemoteRunner(sshConnection, boshLogger), boshLogger),
				))
			})
		})

		Context("when the instance has a named backup artifact and a default artifact", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{jobWithBackupScript1, jobWithBackupScriptAndMetadata})
			})

			It("returns the named artifact and the default artifact", func() {
				Expect(backupArtifacts).To(ConsistOf(
					instance.NewBackupArtifact(
						jobWithBackupScript1,
						deployedInstance,
						instance.NewRemoteRunner(sshConnection, boshLogger),
						boshLogger),
					instance.NewBackupArtifact(
						jobWithBackupScriptAndMetadata,
						deployedInstance,
						instance.NewRemoteRunner(sshConnection, boshLogger),
						boshLogger),
				))
			})
		})

		Context("when the instance has only a named backup artifact", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{jobWithBackupScriptAndMetadata})
			})

			It("returns only the named backup artifact", func() {
				Expect(backupArtifacts).To(Equal(
					[]orchestrator.BackupArtifact{
						instance.NewBackupArtifact(
							jobWithBackupScriptAndMetadata,
							deployedInstance,
							instance.NewRemoteRunner(sshConnection, boshLogger),
							boshLogger,
						),
					},
				))
			})
		})

		Context("when the instance has some jobs with no backup scripts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{jobWithBackupScript1, jobWithOnlyLockScript})
			})

			It("only returns artifacts for the jobs with backup scripts", func() {
				Expect(backupArtifacts).To(Equal(
					[]orchestrator.BackupArtifact{
						instance.NewBackupArtifact(
							jobWithBackupScript1,
							deployedInstance,
							instance.NewRemoteRunner(sshConnection, boshLogger),
							boshLogger,
						),
					},
				))
			})
		})
	})

	Describe("ArtifactsToRestore", func() {
		var restoreArtifacts []orchestrator.BackupArtifact
		var instanceIdentifier instance.InstanceIdentifier

		var jobWithRestoreScript1 = instance.NewJob(remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-restore-script-1/bin/bbr/restore"},
			instance.Metadata{})
		var jobWithRestoreScript2 = instance.NewJob(remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-restore-script-2/bin/bbr/restore"},
			instance.Metadata{})
		var jobWithRestoreScriptAndMetadata = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/job-with-restore-script-and-metadata/bin/bbr/restore",
			},
			instance.Metadata{
				RestoreName: "my-artifact",
			},
		)
		var jobWithBackupScript = instance.NewJob(remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-backup-script-1/bin/bbr/backup"},
			instance.Metadata{})
		var jobWithOnlyLockScript = instance.NewJob(remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-only-lock-script/bin/bbr/pre-restore-lock"},
			instance.Metadata{})

		JustBeforeEach(func() {
			restoreArtifacts = deployedInstance.ArtifactsToRestore()
		})

		BeforeEach(func() {
			instanceIdentifier = instance.InstanceIdentifier{InstanceGroupName: instanceGroupName, InstanceId: instanceID}
		})

		Context("Has no named restore artifacts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					jobWithRestoreScript1,
					jobWithRestoreScript2,
					jobWithBackupScript,
				})
			})

			It("returns the default artifacts", func() {
				Expect(restoreArtifacts).To(ConsistOf(
					instance.NewRestoreArtifact(jobWithRestoreScript1, deployedInstance, instance.NewRemoteRunner(sshConnection, boshLogger), boshLogger),
					instance.NewRestoreArtifact(jobWithRestoreScript2, deployedInstance, instance.NewRemoteRunner(sshConnection, boshLogger), boshLogger),
				))
			})
		})

		Context("Has a named restore artifact", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{jobWithRestoreScript1, jobWithRestoreScriptAndMetadata})
			})

			It("returns the named artifact and the default artifact", func() {
				Expect(restoreArtifacts).To(ConsistOf(
					instance.NewRestoreArtifact(jobWithRestoreScript1, deployedInstance, instance.NewRemoteRunner(sshConnection, boshLogger), boshLogger),
					instance.NewRestoreArtifact(jobWithRestoreScriptAndMetadata, deployedInstance, instance.NewRemoteRunner(sshConnection, boshLogger), boshLogger),
				))
			})
		})

		Context("has only named restore artifacts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{jobWithRestoreScriptAndMetadata})
			})

			It("returns only the named artifact", func() {
				Expect(restoreArtifacts).To(Equal(
					[]orchestrator.BackupArtifact{
						instance.NewRestoreArtifact(jobWithRestoreScriptAndMetadata, deployedInstance, instance.NewRemoteRunner(sshConnection, boshLogger), boshLogger),
					},
				))
			})
		})

		Context("when the instance has some jobs with no restore scripts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{jobWithRestoreScript1, jobWithOnlyLockScript})
			})

			It("only returns artifacts for the jobs with restore scripts", func() {
				Expect(restoreArtifacts).To(Equal(
					[]orchestrator.BackupArtifact{
						instance.NewBackupArtifact(
							jobWithRestoreScript1,
							deployedInstance,
							instance.NewRemoteRunner(sshConnection, boshLogger),
							boshLogger,
						),
					},
				))
			})
		})
	})
})
