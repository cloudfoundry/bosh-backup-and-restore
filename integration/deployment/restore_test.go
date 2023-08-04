package deployment

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockbosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockhttp"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"

	"archive/tar"
	"bytes"

	"path/filepath"

	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restore", func() {
	var director *mockhttp.Server
	var restoreWorkspace string
	var verifyMocks bool
	manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
  - name: redis-writer
    release: redis
- name: redis-server
  instances: 1
  jobs:
  - name: redis
    release: redis
  - name: redis-writer
    release: redis
- name: redis-backup-node
  instances: 1
  jobs:
  - name: redis
    release: redis
- name: redis-restore-node
  instances: 1
  jobs:
  - name: redis
    release: redis
`

	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		restoreWorkspace, err = os.MkdirTemp(".", "restore-workspace-")
		verifyMocks = true
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(restoreWorkspace)).To(Succeed())
		if verifyMocks {
			director.VerifyMocks()
		}
	})

	Context("when deployment is not present", func() {
		var session *gexec.Session
		deploymentName := "my-new-deployment"

		BeforeEach(func() {
			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances: []`))

			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeBasic(),
				mockbosh.VMsForDeployment(deploymentName).NotFound(),
			)
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", "my-new-deployment",
				"restore",
				"--artifact-path", deploymentName)

		})

		It("fails and prints an error", func() {
			By("failing", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(session.Err).To(gbytes.Say("Director responded with non-successful status code"))
			})

			By("not printing the stack trace", func() {
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
			})

			By("writes the stack trace", func() {
				files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
				Expect(err).NotTo(HaveOccurred())
				logFilePath := files[0]
				_, err = os.Stat(logFilePath)
				Expect(os.IsNotExist(err)).To(BeFalse())
				stackTrace, err := os.ReadFile(logFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
			})
		})
	})

	Context("when artifact is not present", func() {
		var session *gexec.Session

		BeforeEach(func() {
			director.VerifyAndMock(mockbosh.Info().WithAuthTypeBasic())
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", "my-new-deployment",
				"restore",
				"--artifact-path", "i-am-not-here")

		})

		It("fails and prints an error", func() {
			By("failing", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(session.Err).To(gbytes.Say("i-am-not-here: no such file or directory"))
			})

			By("not printing the stack trace", func() {
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
			})

			By("writes the stack trace", func() {
				files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
				Expect(err).NotTo(HaveOccurred())
				logFilePath := files[0]
				_, err = os.Stat(logFilePath)
				Expect(os.IsNotExist(err)).To(BeFalse())
				stackTrace, err := os.ReadFile(logFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
			})
		})
	})

	Context("when the backup is corrupted", func() {
		var session *gexec.Session
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(mockbosh.Info().WithAuthTypeBasic())

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
  artifacts:
  - name: redis
    checksums:
      redis-backup: this-is-not-a-checksum-this-is-only-a-tribute`))

			backupContents, err := os.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))},
				"deployment",
				"--debug",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		It("fails and prints an error", func() {
			By("failing", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("logging the steps it takes", func() {
				Expect(session.Out).To(gbytes.Say("INFO - Starting restore of my-new-deployment"))
				Expect(session.Out).To(gbytes.Say("INFO - Validating backup artifact for my-new-deployment"))
			})

			By("printing an error", func() {
				Expect(session.Err).To(gbytes.Say("Backup is corrupted"))
			})

			By("not printing the stack trace", func() {
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
			})

			By("writes the stack trace", func() {
				files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
				Expect(err).NotTo(HaveOccurred())
				logFilePath := files[0]
				_, err = os.Stat(logFilePath)
				Expect(os.IsNotExist(err)).To(BeFalse())
				stackTrace, err := os.ReadFile(logFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
			})
		})
	})

	Context("when deployment has a single instance", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string
		var waitForRestoreToFinish bool
		var stdin io.WriteCloser

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
						ID:      "fake-uuid",
						Index:   newIndex(0),
					}}),
				DownloadManifest(deploymentName, manifest),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSH(deploymentName, "redis-dedicated-node"))...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run
`)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/pre-restore-lock-script-was-run
`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
  artifacts:
  - name: redis
    checksums:
      ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

			backupContents, err := os.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar",
				backupContents)
			waitForRestoreToFinish = true
		})

		JustBeforeEach(func() {
			env := []string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))}

			params := []string{
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName,
			}

			if waitForRestoreToFinish {
				session = binary.Run(
					restoreWorkspace,
					env,
					params...,
				)
			} else {
				session, stdin = binary.Start(
					restoreWorkspace,
					env,
					params...,
				)
				Eventually(session).Should(gbytes.Say(".+"))
			}
		})

		AfterEach(func() {
			instance1.DieInBackground()
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
		})

		Context("and the restore script works", func() {
			BeforeEach(func() {
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)
			})

			It("runs the restore script successfully and cleans up", func() {
				By("succeeding", func() {
					Expect(session.ExitCode()).To(Equal(0))
				})

				By("logging the steps it takes", func() {
					Expect(session.Out).To(gbytes.Say("INFO - Starting restore of my-new-deployment"))
					Expect(session.Out).To(gbytes.Say("INFO - Validating backup artifact for my-new-deployment"))
					Expect(session.Out).To(gbytes.Say("INFO - Looking for scripts"))
					Expect(session.Out).To(gbytes.Say("INFO - Copying backup -- 12K uncompressed -- for job redis on redis-dedicated-node/0..."))
					Expect(session.Out).To(gbytes.Say("INFO - Finished copying backup for job redis on redis-dedicated-node/0."))
					Expect(session.Out).To(gbytes.Say("INFO - Completed restore of my-new-deployment"))
				})

				By("cleaning up the archive file on the remote", func() {
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup/redis-backup")).To(BeFalse())
				})

				By("running the restore script on the remote", func() {
					Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
					Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
				})

				By("running the pre-restore-lock script on the remote", func() {
					Expect(instance1.FileExists("/tmp/pre-restore-lock-script-was-run")).To(BeTrue())
				})

				By("running the post-backup-unlock script on the remote", func() {
					Expect(instance1.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
				})
			})

			Context("and the bbr process receives SIGINT while restoring", func() {
				BeforeEach(func() {
					waitForRestoreToFinish = false

					By("creating a restore script that takes a while")
					instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh

set -u

sleep 2
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)
				})

				Context("and the user decides to cancel the restore", func() {
					BeforeEach(func() {
						verifyMocks = false
					})

					It("terminates", func() {
						session.Interrupt()

						By("not terminating", func() {
							Consistently(session.Exited).ShouldNot(BeClosed(), "bbr exited without user confirmation")
						})

						By("outputting a helpful message", func() {
							Eventually(session).Should(gbytes.Say(`Stopping a restore can leave the system in bad state. Are you sure you want to cancel\? \[yes/no\]`))
						})

						By("buffering the logs", func() {
							Expect(string(session.Out.Contents())).To(HaveSuffix("[yes/no]\n"))
						})

						stdin.Write([]byte("yes\n"))

						By("waiting for the restore to finish successfully", func() {
							Eventually(session, 10).Should(gexec.Exit(1))
						})

						By("outputting a warning about cleanup", func() {
							Eventually(session).Should(gbytes.Say("It is recommended that you run `bbr restore-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."))
						})

						By("not completing the restore", func() {
							Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeFalse())
						})
					})
				})

				Context("and the user decides not to to cancel the restore", func() {
					It("continues to run", func() {
						session.Interrupt()

						By("not terminating", func() {
							Consistently(session.Exited).ShouldNot(BeClosed(), "bbr exited without user confirmation")
						})

						By("outputting a helpful message", func() {
							Eventually(session).Should(gbytes.Say(`Stopping a restore can leave the system in bad state. Are you sure you want to cancel\? \[yes/no\]`))
						})

						By("buffering the logs", func() {
							Expect(string(session.Out.Contents())).To(HaveSuffix("[yes/no]\n"))
						})

						stdin.Write([]byte("no\n"))

						By("waiting for the restore to finish successfully", func() {
							Eventually(session, 10).Should(gexec.Exit(0))
						})

						By("still completing the restore", func() {
							Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
							Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
						})

						By("should output buffered logs", func() {
							Expect(string(session.Out.Contents())).NotTo(HaveSuffix(fmt.Sprintf("[yes/no]\n")))
						})
					})
				})
			})

			Context("and pre-restore-lock script fails", func() {
				BeforeEach(func() {
					instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
	>&2 echo "dear lord"; exit 1`)
				})
				It("exits cleanly", func() {
					By("not running restore", func() {
						Expect(instance1.FileExists("/tmp/restore-script-was-run")).NotTo(BeTrue())
					})

					By("running the post-restore-unlock script on the remote", func() {
						Expect(instance1.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
					})

				})
			})
		})

		Context("when restore fails", func() {
			BeforeEach(func() {
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
	>&2 echo "dear lord"; exit 1`)
			})

			It("fails and returns the failure", func() {
				By("failing", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("returning the failure", func() {
					Expect(session.Err).To(gbytes.Say("dear lord"))
				})
				By("not printing the stack trace", func() {
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
				})

				By("writes the stack trace", func() {
					files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
					Expect(err).NotTo(HaveOccurred())
					logFilePath := files[0]
					_, err = os.Stat(logFilePath)
					Expect(os.IsNotExist(err)).To(BeFalse())
					stackTrace, err := os.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})

				By("running the post-restore-unlock script on the remote", func() {
					Expect(instance1.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
				})
			})
		})

		Context("when the backup artifact already exists", func() {
			BeforeEach(func() {
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)
				instance1.CreateDir("/var/vcap/store/bbr-backup")
			})

			It("fails, returns an error and does not delete the artifact", func() {
				By("failing", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("returning the correct error", func() {
					Expect(session.Err).To(gbytes.Say(
						"Directory /var/vcap/store/bbr-backup already exists on instance redis-dedicated-node/fake-uuid",
					))
				})

				By("not printing the stack trace", func() {
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
				})

				By("writes the stack trace", func() {
					files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
					Expect(err).NotTo(HaveOccurred())
					logFilePath := files[0]
					_, err = os.Stat(logFilePath)
					Expect(os.IsNotExist(err)).To(BeFalse())
					stackTrace, err := os.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})

				By("not deleting the artifact", func() {
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
				})
			})
		})

		Context("when the job is disabled", func() {
			BeforeEach(func() {
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
skip_bbr_scripts: true
"`)
			})

			It("reports that no restore scripts were found", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
				Expect(string(session.Err.Contents())).To(ContainSubstring("has no restore scripts"))
			})
		})
	})

	Context("when deployment has a multiple instances", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var instance2 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			instance2 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
						Index:   newIndex(0),
						ID:      "fake-uuid",
					},
					{
						IPs:     []string{"10.0.0.10"},
						JobName: "redis-server",
						Index:   newIndex(0),
						ID:      "fake-uuid",
					}}),
				DownloadManifest(deploymentName, manifest),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				SetupSSH(deploymentName, "redis-server", "fake-uuid", 0, instance2),
				CleanupSSH(deploymentName, "redis-dedicated-node"),
				CleanupSSH(deploymentName, "redis-server"))...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)
			instance2.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
  artifacts:
  - name: redis
    checksums:
      ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0
- name: redis-server
  index: 0
  artifacts:
  - name: redis
    checksums:
      ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

			backupContents, err := os.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-server-0-redis.tar", backupContents)
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		AfterEach(func() {
			instance1.DieInBackground()
			instance2.DieInBackground()
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
		})

		It("runs the restore script and cleans up", func() {
			By("succeeding", func() {
				Expect(session.ExitCode()).To(Equal(0))
			})

			By("cleaning up the archive file on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup/redis-backup")).To(BeFalse())
				Expect(instance2.FileExists("/var/vcap/store/bbr-backup/redis-backup")).To(BeFalse())
			})

			By("running the restore script on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
				Expect(instance2.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				Expect(instance2.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
			})
		})

		Context("and one of the jobs is disabled", func() {
			BeforeEach(func() {
				instance2.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
skip_bbr_scripts: true
"`)
			})

			It("does not restore the disable job", func() {
				Expect(session.ExitCode()).To(BeZero())
				Expect(string(session.Out.Contents())).To(ContainSubstring("Found disabled jobs on instance redis-server/fake-uuid jobs: redis"))
				Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
				Expect(instance2.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeFalse())
				Expect(instance2.FileExists("/tmp/restore-script-was-run")).To(BeFalse())
			})
		})

		Context("with ordering on pre-restore-lock (where the default order would be wrong)", func() {
			BeforeEach(func() {
				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/redis-pre-restore-lock-called
exit 0`)
				instance2.CreateScript(
					"/var/vcap/jobs/redis-writer/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/redis-writer-pre-restore-lock-called
exit 0`)
				instance2.CreateScript("/var/vcap/jobs/redis-writer/bin/bbr/metadata",
					`#!/usr/bin/env sh
echo "---
restore_should_be_locked_before:
- job_name: redis
  release: redis
"`)
			})

			It("locks in the right order", func() {
				redisLockTime := instance1.GetCreatedTime("/tmp/redis-pre-restore-lock-called")
				redisWriterLockTime := instance2.GetCreatedTime("/tmp/redis-writer-pre-restore-lock-called")

				Expect(session.Out).To(gbytes.Say("Detected order: redis-writer should be locked before redis/redis during restore"))

				Expect(redisWriterLockTime < redisLockTime).To(BeTrue(), fmt.Sprintf(
					"Writer locked at %s, which is after the server locked (%s)",
					strings.TrimSuffix(redisWriterLockTime, "\n"),
					strings.TrimSuffix(redisLockTime, "\n")))

			})
		})

		Context("with ordering on pre-restore-lock (where the default ordering would unlock in the wrong order)",
			func() {
				BeforeEach(func() {
					instance2.CreateScript(
						"/var/vcap/jobs/redis/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/redis-pre-restore-lock-called
exit 0`)
					instance1.CreateScript(
						"/var/vcap/jobs/redis-writer/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/redis-writer-pre-restore-lock-called
exit 0`)
					instance2.CreateScript(
						"/var/vcap/jobs/redis/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/redis-post-restore-unlock-called
exit 0`)
					instance1.CreateScript(
						"/var/vcap/jobs/redis-writer/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/redis-writer-post-restore-unlock-called
exit 0`)
					instance1.CreateScript("/var/vcap/jobs/redis-writer/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
restore_should_be_locked_before:
- job_name: redis
  release: redis
"`)
				})

				It("unlocks in the right order", func() {
					By("unlocking the redis job before unlocking the redis-writer job")
					redisUnlockTime := instance2.GetCreatedTime("/tmp/redis-post-restore-unlock-called")
					redisWriterUnlockTime := instance1.GetCreatedTime("/tmp/redis-writer-post-restore-unlock-called")

					Expect(redisUnlockTime < redisWriterUnlockTime).To(BeTrue(), fmt.Sprintf(
						"Writer unlocked at %s, which is before the server unlocked (%s)",
						strings.TrimSuffix(redisWriterUnlockTime, "\n"),
						strings.TrimSuffix(redisUnlockTime, "\n")))
				})
			})

		Context("but the pre-restore-lock ordering is cyclic", func() {
			BeforeEach(func() {
				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/redis-pre-restore-lock-called
exit 0`)
				instance1.CreateScript(
					"/var/vcap/jobs/redis-writer/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/redis-writer-pre-restore-lock-called
exit 0`)
				instance1.CreateScript("/var/vcap/jobs/redis-writer/bin/bbr/metadata",
					`#!/usr/bin/env sh
echo "---
restore_should_be_locked_before:
- job_name: redis
  release: redis
"`)
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata",
					`#!/usr/bin/env sh
echo "---
restore_should_be_locked_before:
- job_name: redis-writer
  release: redis
"`)
			})

			It("Should fail", func() {
				By("exiting with an error", func() {
					Expect(session).To(gexec.Exit(1))
				})

				By("printing a helpful error message", func() {
					Expect(session.Err).To(gbytes.Say("job locking dependency graph is cyclic"))
				})
			})
		})
	})

	Context("when deployment has named artifacts", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
			instance1.DieInBackground()
		})

		Context("and the job name is not special", func() {
			BeforeEach(func() {
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					VmsForDeployment(deploymentName, []mockbosh.VMsOutput{

						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
							ID:      "fake-uuid",
							Index:   newIndex(0),
						}}),
					DownloadManifest(deploymentName, manifest),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"))...)
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
restore_name: foo
"`)
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)

				Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
				createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
custom_artifacts:
- name: foo
  checksums:
    ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

				backupContents, err := os.ReadFile("../../fixtures/backup.tar")
				Expect(err).NotTo(HaveOccurred())
				createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tar", backupContents)

				createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", createTarWithContents(map[string]string{}))
			})

			It("runs the restore script and cleans up", func() {
				By("fails", func() {
					Expect(session.ExitCode()).NotTo(Equal(0))
				})

				By("returning the failure", func() {
					Expect(session.Out).To(gbytes.Say("ERROR - discontinued metadata keys backup_name/restore_name found on instance redis-dedicated-node. bbr cannot restore this backup artifact."))
					Expect(session.Err).To(gbytes.Say("discontinued metadata keys backup_name/restore_name found on instance redis-dedicated-node. bbr cannot restore this backup artifact."))
				})

				By("not running the restore script on the remote", func() {
					Expect(instance1.FileExists("/var/vcap/store/redis-server" +
						"/redis-backup")).To(BeFalse())
					Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeFalse())
				})
			})
		})

		Context("and the job is called mysql-restore", func() {
			const instanceGroupName = "mysql"

			BeforeEach(func() {
				manifest = `---
instance_groups:
- name: mysql
  instances: 1
  jobs:
  - name: redis
    release: redis
  - name: mysql-restore
    release: cf-backup-and-restore
`
				director.VerifyAndMock(AppendBuilders(
					InfoWithBasicAuth(),
					VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: instanceGroupName,
							ID:      "fake-uuid",
							Index:   newIndex(0),
						},
					}),
					DownloadManifest(deploymentName, manifest),
					SetupSSH(deploymentName, instanceGroupName, "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, instanceGroupName))...)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/redis-restore-script-was-run`)

				instance1.CreateScript("/var/vcap/jobs/mysql-restore/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
restore_name: mysql-artifact
"`)
				instance1.CreateScript("/var/vcap/jobs/mysql-restore/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/mysql-restore
touch /tmp/mysql-restore-script-was-run`)

				Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
				createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: mysql
  index: 0
  artifacts:
  - name: redis
    checksums:
      ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0
custom_artifacts:
- name: mysql-artifact
  checksums: {}
`))

				backupContents, err := os.ReadFile("../../fixtures/backup.tar")
				Expect(err).NotTo(HaveOccurred())

				createFileWithContents(
					restoreWorkspace+"/"+deploymentName+"/"+"mysql-0-redis.tar",
					backupContents,
				)
			})

			It("ignores the mysql-restore job scripts", func() {
				By("succeeding", func() {
					Expect(session.ExitCode()).To(Equal(0))
				})

				By("not printing a warning", func() {
					Expect(string(session.Out.Contents())).NotTo(ContainSubstring("discontinued metadata keys backup_name/restore_name"))
				})

				By("cleaning up the archive file on the remote", func() {
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
				})

				By("running the redis restore script on the remote", func() {
					Expect(instance1.FileExists("/tmp/redis-restore-script-was-run")).To(BeTrue())
					Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				})

				By("not running the mysql restore script on the remote", func() {
					Expect(instance1.FileExists("/tmp/mysql-restore-script-was-run")).To(BeFalse())
				})
			})
		})
	})

	Context("when deployment has named artifacts using backup_one_restore_all", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var instance2 *testcluster.Instance
		var deploymentName string

		manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 2
  jobs:
  - name: redis-dedicated-node
    release: redis
    properties:
      bbr:
        backup_one_restore_all: true
  - name: redis-writer
    release: redis
  - name: redis-broker
    release: redis
`

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			instance2 = testcluster.NewInstance()

			deploymentName = "my-new-deployment"
			twoInstancesInSameGroupResponse := func(instanceGroupName string) []mockbosh.VMsOutput {
				return []mockbosh.VMsOutput{
					{
						IPs:       []string{instance1.Address()},
						JobName:   instanceGroupName,
						Index:     newIndex(0),
						ID:        "fake-uuid-0",
						Bootstrap: true,
					},
					{
						IPs:     []string{instance2.Address()},
						JobName: instanceGroupName,
						Index:   newIndex(1),
						ID:      "fake-uuid-1",
					},
				}
			}

			MockDirectorWith(director,
				mockbosh.Info().WithAuthTypeBasic(),
				VmsForDeployment(deploymentName, twoInstancesInSameGroupResponse("redis-dedicated-node")),
				DownloadManifest(deploymentName, manifest),
				SetupSSHForAllInstances(deploymentName, "redis-dedicated-node", twoInstancesInSameGroupResponse("redis-dedicated-node"), []*testcluster.Instance{
					instance1, instance2,
				}),
				append(
					CleanupSSH(deploymentName, "redis-dedicated-node"),
					CleanupSSH(deploymentName, "redis-dedicated-node")...),
			)
			instance1.CreateScript("/var/vcap/jobs/redis-dedicated-node/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)

			instance2.CreateScript("/var/vcap/jobs/redis-dedicated-node/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: "1"
  artifacts:
  - name: redis-dedicated-node
    checksums: {}
custom_artifacts:
- name: redis-dedicated-node-redis-backup-one-restore-all
  checksums:
    ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0
backup_activity:
  start_time: 2019/02/27 10:10:30 GMT
  finish_time: 2019/02/27 10:10:30 GMT`))

			backupContents, err := os.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-redis-backup-one-restore-all.tar", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-1-redis-dedicated-node.tar", createTarWithContents(map[string]string{}))
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
			instance1.DieInBackground()
		})

		It("runs the restore script and cleans up", func() {
			By("succeeding", func() {
				Expect(session.ExitCode()).To(Equal(0))
			})

			By("cleaning up the archive file on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
			})

			By("running the restore script on the remote", func() {
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
				Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())

				Expect(instance2.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
				Expect(instance2.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
			})
		})
	})

	Context("when the backup with named artifacts on disk is corrupted", func() {
		var session *gexec.Session
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "my-new-deployment"

			director.VerifyAndMock(mockbosh.Info().WithAuthTypeBasic())
			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-backup-node
  index: 0
custom_artifacts:
- name: foo
  checksums:
    ./redis/redis-backup: this-is-damn-wrong`))

			backupContents, err := os.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tar", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-backup-node-0.tar", createTarWithContents(map[string]string{}))
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		It("fails", func() {
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Err).To(gbytes.Say("Backup is corrupted"))
		})

		It("writes the stack trace", func() {
			files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
			Expect(err).NotTo(HaveOccurred())
			logFilePath := files[0]
			_, err = os.Stat(logFilePath)
			Expect(os.IsNotExist(err)).To(BeFalse())
			stackTrace, err := os.ReadFile(logFilePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
		})
	})

	Context("the cleanup fails", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
						ID:      "fake-uuid",
						Index:   newIndex(0),
					}}),
				DownloadManifest(deploymentName, manifest),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSHFails(deploymentName, "redis-dedicated-node", "cleanup err"))...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set-u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server/
touch /tmp/restore-script-was-run`)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
  artifacts:
  - name: redis
    checksums:
      ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

			backupContents, err := os.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		AfterEach(func() {
			instance1.DieInBackground()
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
		})

		It("runs the restore script, fails and cleans up", func() {
			By("failing", func() {
				Expect(session.ExitCode()).To(Equal(16))
			})

			By("cleaning up the archive file on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup/redis-backup")).To(BeFalse())
			})

			By("running the restore script on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
			})

			By("running the post-restore-unlock scripts", func() {
				Expect(instance1.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
			})

			By("returning the failure", func() {
				Expect(session.Err).To(gbytes.Say("cleanup err"))
			})

			By("not printing the stack trace", func() {
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
			})
		})
	})
})

func createFileWithContents(filePath string, contents []byte) {
	file, err := os.Create(filePath)
	Expect(err).NotTo(HaveOccurred())
	_, err = file.Write([]byte(contents))
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())
}

func createTarWithContents(files map[string]string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	tarFile := tar.NewWriter(bytesBuffer)

	for filename, contents := range files {
		hdr := &tar.Header{
			Name: filename,
			Mode: 0600,
			Size: int64(len(contents)),
		}
		if err := tarFile.WriteHeader(hdr); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		if _, err := tarFile.Write([]byte(contents)); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}
	if err := tarFile.Close(); err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
	Expect(tarFile.Close()).NotTo(HaveOccurred())
	return bytesBuffer.Bytes()
}
