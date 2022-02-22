package instance_test

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	sshfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("DeployedInstance", func() {
	var boshLogger boshlog.Logger
	var logOutput *gbytes.Buffer
	var instanceGroupName, instanceIndex, instanceID string
	var jobs orchestrator.Jobs
	var remoteRunner *sshfakes.FakeRemoteRunner

	var deployedInstance *instance.DeployedInstance
	BeforeEach(func() {
		instanceGroupName = "instance-group-name"
		instanceIndex = "instance-index"
		instanceID = "instance-id"
		logOutput = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(logOutput, "[bosh-package] ", log.Lshortfile))
		remoteRunner = new(sshfakes.FakeRemoteRunner)
	})

	JustBeforeEach(func() {
		remoteRunner.ConnectedUsernameReturns("sshUsername")
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
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/dave/bin/bbr/backup",
						},
						instance.Metadata{},
						false,
						false,
					),
				})
			})

			It("returns true", func() {
				Expect(actualBackupable).To(BeTrue())
			})
		})

		Describe("there are no backup scripts in the job directories", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/dave/bin/foo",
						},
						instance.Metadata{},
						false,
						false,
					),
				})
			})

			It("returns false", func() {
				Expect(actualBackupable).To(BeFalse())
			})
		})
	})

	Describe("ArtifactDirExists", func() {
		var dirExists bool

		JustBeforeEach(func() {
			dirExists, _ = deployedInstance.ArtifactDirExists()
		})

		It("queries whether the artifact directory is present", func() {
			Expect(remoteRunner.DirectoryExistsCallCount()).To(Equal(1))
			Expect(remoteRunner.DirectoryExistsArgsForCall(0)).To(Equal("/var/vcap/store/bbr-backup"))
		})

		Context("when artifact directory does not exist", func() {
			BeforeEach(func() {
				remoteRunner.DirectoryExistsReturns(false, nil)
			})

			It("returns false", func() {
				Expect(dirExists).To(BeFalse())
			})
		})

		Context("when artifact directory does exist", func() {
			BeforeEach(func() {
				remoteRunner.DirectoryExistsReturns(true, nil)
			})

			It("returns true", func() {
				Expect(dirExists).To(BeTrue())
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
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/dave/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
				})
			})

			It("returns true", func() {
				Expect(actualRestorable).To(BeTrue())
			})
		})

		Describe("there are no restore scripts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/dave/bin/foo",
						},
						instance.Metadata{},
						false,
						false,
					),
				})
			})

			It("returns false", func() {
				Expect(actualRestorable).To(BeFalse())
			})
		})
	})

	Describe("HasMetadataRestoreNames", func() {
		Context("when a job has metadata restore name", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/dave/bin/foo",
						},
						instance.Metadata{
							RestoreName: "foo",
						},
						false,
						false,
					),
				})
			})

			It("returns true", func() {
				Expect(deployedInstance.HasMetadataRestoreNames()).To(BeTrue())
			})
		})

		Context("when none of the jobs has metadata restore name", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/dave/bin/foo",
						},
						instance.Metadata{},
						false,
						false,
					),
				})
			})

			It("returns false", func() {
				Expect(deployedInstance.HasMetadataRestoreNames()).To(BeFalse())

			})
		})
	})

	Describe("Jobs", func() {
		BeforeEach(func() {
			jobs = orchestrator.Jobs([]orchestrator.Job{
				instance.NewJob(
					remoteRunner,
					instanceGroupName+"/"+instanceID,
					boshLogger,
					"",
					instance.BackupAndRestoreScripts{
						"/var/vcap/jobs/dave/bin/foo",
					},
					instance.Metadata{},
					false,
					false,
				),
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
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/foo/bin/bbr/backup",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/bar/bin/bbr/backup",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/baz/bin/bbr/backup",
						},
						instance.Metadata{},
						false,
						false,
					),
				})
			})

			It("uses the remote runner to create each job's backup folder and run each backup script providing the "+
				"correct ARTIFACT_DIRECTORY and BBR_ARTIFACT_DIRECTORY", func() {
				Expect(remoteRunner.CreateDirectoryCallCount()).To(Equal(3))
				Expect(remoteRunner.RunScriptWithEnvCallCount()).To(Equal(3))
				Expect([]string{
					remoteRunner.CreateDirectoryArgsForCall(0),
					remoteRunner.CreateDirectoryArgsForCall(1),
					remoteRunner.CreateDirectoryArgsForCall(2),
				}).To(ConsistOf(
					"/var/vcap/store/bbr-backup/foo",
					"/var/vcap/store/bbr-backup/bar",
					"/var/vcap/store/bbr-backup/baz",
				))

				specifiedScriptPath, specifiedEnvVars, _, _ := remoteRunner.RunScriptWithEnvArgsForCall(0)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/foo/bin/bbr/backup"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/foo/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/foo/",
				}))

				specifiedScriptPath, specifiedEnvVars, _, _ = remoteRunner.RunScriptWithEnvArgsForCall(1)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/bar/bin/bbr/backup"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/bar/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/bar/",
				}))

				specifiedScriptPath, specifiedEnvVars, _, _ = remoteRunner.RunScriptWithEnvArgsForCall(2)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/baz/bin/bbr/backup"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/baz/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/baz/",
				}))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(logOutput.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/backup`))
				Expect(string(logOutput.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/backup`))
				Expect(string(logOutput.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/backup`))
				Expect(string(logOutput.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is backing up the job on the instance", func() {
				Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up foo on %s/%s",
					instanceGroupName,
					instanceID,
				)))
				Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up bar on %s/%s",
					instanceGroupName,
					instanceID,
				)))
				Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up baz on %s/%s",
					instanceGroupName,
					instanceID,
				)))
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
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/foo/bin/bbr/backup",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"dave",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/baz/bin/bbr/backup",
						},
						instance.Metadata{},
						true,
						true,
					),
				})
			})

			It("uses the remote runner to create each job's backup folder and run each backup script providing the "+
				"correct BBR_ARTIFACT_DIRECTORY and ARTIFACT_DIRECTORY", func() {

				Expect(remoteRunner.CreateDirectoryCallCount()).To(Equal(2))
				Expect(remoteRunner.RunScriptWithEnvCallCount()).To(Equal(2))
				Expect([]string{
					remoteRunner.CreateDirectoryArgsForCall(0),
					remoteRunner.CreateDirectoryArgsForCall(1),
				}).To(ConsistOf(
					"/var/vcap/store/bbr-backup/foo",
					"/var/vcap/store/bbr-backup/baz-dave-backup-one-restore-all",
				))
				specifiedScriptPath, specifiedEnvVars, _, _ := remoteRunner.RunScriptWithEnvArgsForCall(0)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/foo/bin/bbr/backup"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/foo/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/foo/",
				}))

				specifiedScriptPath, specifiedEnvVars, _, _ = remoteRunner.RunScriptWithEnvArgsForCall(1)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/baz/bin/bbr/backup"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/baz-dave-backup-one-restore-all/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/baz-dave-backup-one-restore-all/",
				}))

			})
		})

		Context("when there are multiple jobs with no backup scripts", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/foo/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/bar/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
				})
			})
			It("doesn't make calls to the instance over the remote runner", func() {
				Expect(remoteRunner.Invocations()).To(HaveLen(0))
			})
		})

		Context("when there are several scripts and two of them cause an error", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/foo/bin/bbr/backup",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/bar/bin/bbr/backup",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/baz/bin/bbr/backup",
						},
						instance.Metadata{},
						false,
						false,
					),
				})

				remoteRunner.RunScriptWithEnvStub = func(cmd string, envVars map[string]string, label string, stdout io.Writer) error {
					if strings.Contains(cmd, "jobs/bar") {
						return fmt.Errorf("no space left on device")
					} else if strings.Contains(cmd, "jobs/baz") {
						return fmt.Errorf("huge failure")
					} else {
						return nil
					}
				}
			})

			It("fails", func() {
				By("including all relevant information", func() {
					Expect(err).To(MatchError(SatisfyAll(
						ContainSubstring(fmt.Sprintf("Error attempting to run backup for job bar on %s/%s", instanceGroupName, instanceID)),
						ContainSubstring(fmt.Sprintf("Error attempting to run backup for job baz on %s/%s", instanceGroupName, instanceID)),
						ContainSubstring("no space left on device"),
						ContainSubstring("huge failure"),
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

	Describe("Restore", func() {
		var actualError error

		JustBeforeEach(func() {
			actualError = deployedInstance.Restore()
		})

		Context("when there are multiple restore scripts in multiple job directories", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/foo/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/bar/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/baz/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
				})
			})

			It("uses the remote runner to run each restore script providing the correct ARTIFACT_DIRECTORY", func() {
				Expect(remoteRunner.RunScriptWithEnvCallCount()).To(Equal(3))

				specifiedScriptPath, specifiedEnvVars, _ , _:= remoteRunner.RunScriptWithEnvArgsForCall(0)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/foo/bin/bbr/restore"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/foo/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/foo/",
				}))

				specifiedScriptPath, specifiedEnvVars, _, _ = remoteRunner.RunScriptWithEnvArgsForCall(1)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/bar/bin/bbr/restore"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/bar/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/bar/",
				}))

				specifiedScriptPath, specifiedEnvVars, _, _= remoteRunner.RunScriptWithEnvArgsForCall(2)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/baz/bin/bbr/restore"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/baz/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/baz/",
				}))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(logOutput.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/restore`))
				Expect(string(logOutput.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/restore`))
				Expect(string(logOutput.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/restore`))
				Expect(string(logOutput.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is restoring a job on the instance", func() {
				Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring foo on %s/%s",
					instanceGroupName,
					instanceID,
				)))

				Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring bar on %s/%s",
					instanceGroupName,
					instanceID,
				)))
				Expect(string(logOutput.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring baz on %s/%s",
					instanceGroupName,
					instanceID,
				)))
			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple restore scripts and one of them is named", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/foo/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/bar/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/baz/bin/bbr/restore",
						},
						instance.Metadata{RestoreName: "special-backup"},
						false,
						false,
					),
				})
			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("uses the remote runner to create each job's backup folder and run each backup script providing the correct BBR_ARTIFACT_DIRECTORY and ARTIFACT_DIRECTORY", func() {
				Expect(remoteRunner.RunScriptWithEnvCallCount()).To(Equal(3))

				specifiedScriptPath, specifiedEnvVars, _ , _ := remoteRunner.RunScriptWithEnvArgsForCall(0)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/foo/bin/bbr/restore"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/foo/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/foo/",
				}))

				specifiedScriptPath, specifiedEnvVars, _, _ = remoteRunner.RunScriptWithEnvArgsForCall(1)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/bar/bin/bbr/restore"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/bar/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/bar/",
				}))

				specifiedScriptPath, specifiedEnvVars, _ , _ = remoteRunner.RunScriptWithEnvArgsForCall(2)
				Expect(specifiedScriptPath).To(Equal("/var/vcap/jobs/baz/bin/bbr/restore"))
				Expect(specifiedEnvVars).To(Equal(map[string]string{
					"ARTIFACT_DIRECTORY":     "/var/vcap/store/bbr-backup/special-backup/",
					"BBR_ARTIFACT_DIRECTORY": "/var/vcap/store/bbr-backup/special-backup/",
				}))
			})
		})

		Context("when there are several scripts and two of them cause an error", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/foo/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/bar/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
					instance.NewJob(
						remoteRunner,
						instanceGroupName+"/"+instanceID,
						boshLogger,
						"",
						instance.BackupAndRestoreScripts{
							"/var/vcap/jobs/baz/bin/bbr/restore",
						},
						instance.Metadata{},
						false,
						false,
					),
				})

				remoteRunner.RunScriptWithEnvStub = func(cmd string, envVars map[string]string, label string, stdout io.Writer) error {
					if strings.Contains(cmd, "jobs/bar") {
						return fmt.Errorf("no space left on device")
					} else if strings.Contains(cmd, "jobs/baz") {
						return fmt.Errorf("huge failure")
					} else {
						return nil
					}
				}
			})

			It("fails", func() {
				By("including all relevant information", func() {
					Expect(actualError).To(MatchError(SatisfyAll(
						ContainSubstring(fmt.Sprintf("Error attempting to run restore for job bar on %s/%s", instanceGroupName, instanceID)),
						ContainSubstring(fmt.Sprintf("Error attempting to run restore for job baz on %s/%s", instanceGroupName, instanceID)),
						ContainSubstring("no space left on device"),
						ContainSubstring("huge failure"),
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

		var jobWithBackupScript1 = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-backup-script-1/bin/bbr/backup"},
			instance.Metadata{},
			false,
			false,
		)
		var jobWithBackupScript2 = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-backup-script-2/bin/bbr/backup"},
			instance.Metadata{},
			false,
			false,
		)
		var jobWithBackupScriptAndMetadata = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			instance.BackupAndRestoreScripts{"/var/vcap/jobs/job-with-backup-script-and-metadata/bin/bbr/backup"},
			instance.Metadata{BackupName: "my-artifact"},
			false,
			false,
		)
		var jobWithRestoreScript = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-restore-script-1/bin/bbr/restore"},
			instance.Metadata{},
			false,
			false,
		)
		var jobWithOnlyLockScript = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-only-lock-script/bin/bbr/pre-backup-lock"},
			instance.Metadata{},
			false,
			false,
		)

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
					instance.NewBackupArtifact(jobWithBackupScript1, deployedInstance, remoteRunner, boshLogger),
					instance.NewBackupArtifact(jobWithBackupScript2, deployedInstance, remoteRunner, boshLogger),
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
						remoteRunner,
						boshLogger),
					instance.NewBackupArtifact(
						jobWithBackupScriptAndMetadata,
						deployedInstance,
						remoteRunner,
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
							remoteRunner,
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
							remoteRunner,
							boshLogger,
						),
					},
				))
			})
		})
	})

	Describe("ArtifactsToRestore", func() {
		var restoreArtifacts []orchestrator.BackupArtifact

		var jobWithRestoreScript1 = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-restore-script-1/bin/bbr/restore"},
			instance.Metadata{},
			false,
			false,
		)
		var jobWithRestoreScript2 = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-restore-script-2/bin/bbr/restore"},
			instance.Metadata{},
			false,
			false,
		)
		var jobWithRestoreScriptAndMetadata = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			instance.BackupAndRestoreScripts{"/var/vcap/jobs/job-with-restore-script-and-metadata/bin/bbr/restore"},
			instance.Metadata{RestoreName: "my-artifact"},
			false,
			false,
		)
		var jobWithBackupScript = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-backup-script-1/bin/bbr/backup"},
			instance.Metadata{},
			false,
			false,
		)
		var jobWithOnlyLockScript = instance.NewJob(
			remoteRunner,
			instanceGroupName+"/"+instanceID,
			boshLogger,
			"",
			[]instance.Script{"/var/vcap/jobs/job-with-only-lock-script/bin/bbr/pre-restore-lock"},
			instance.Metadata{},
			false,
			false,
		)

		JustBeforeEach(func() {
			restoreArtifacts = deployedInstance.ArtifactsToRestore()
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
					instance.NewRestoreArtifact(jobWithRestoreScript1, deployedInstance, remoteRunner, boshLogger),
					instance.NewRestoreArtifact(jobWithRestoreScript2, deployedInstance, remoteRunner, boshLogger),
				))
			})
		})

		Context("Has a named restore artifact", func() {
			BeforeEach(func() {
				jobs = orchestrator.Jobs([]orchestrator.Job{jobWithRestoreScript1, jobWithRestoreScriptAndMetadata})
			})

			It("returns the named artifact and the default artifact", func() {
				Expect(restoreArtifacts).To(ConsistOf(
					instance.NewRestoreArtifact(jobWithRestoreScript1, deployedInstance, remoteRunner, boshLogger),
					instance.NewRestoreArtifact(jobWithRestoreScriptAndMetadata, deployedInstance, remoteRunner, boshLogger),
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
						instance.NewRestoreArtifact(jobWithRestoreScriptAndMetadata, deployedInstance, remoteRunner, boshLogger),
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
							remoteRunner,
							boshLogger,
						),
					},
				))
			})
		})
	})
})
