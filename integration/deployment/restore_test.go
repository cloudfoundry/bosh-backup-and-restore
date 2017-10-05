package deployment

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"

	"archive/tar"
	"bytes"

	"path/filepath"

	"io"
	"time"

	"strings"

	. "github.com/onsi/ginkgo"
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
		restoreWorkspace, err = ioutil.TempDir(".", "restore-workspace-")
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
				[]string{"BOSH_CLIENT_SECRET=admin"},
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
				Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
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
				stackTrace, err := ioutil.ReadFile(logFilePath)
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
				[]string{"BOSH_CLIENT_SECRET=admin"},
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
				Expect(string(session.Err.Contents())).To(ContainSubstring("i-am-not-here: no such file or directory"))
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
				stackTrace, err := ioutil.ReadFile(logFilePath)
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

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
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
				Expect(string(session.Err.Contents())).To(ContainSubstring("Backup is corrupted"))
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
				stackTrace, err := ioutil.ReadFile(logFilePath)
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
						JobID:   "fake-uuid",
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

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar",
				backupContents)
			waitForRestoreToFinish = true
		})

		JustBeforeEach(func() {
			env := []string{"BOSH_CLIENT_SECRET=admin"}

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
							time.Sleep(time.Millisecond * 100) // without this sleep, the following assertion won't ever fail, even if the session does exit
							Expect(session.Exited).NotTo(BeClosed(), "bbr process terminated in response to signal")
						})

						By("outputting a helpful message", func() {
							Eventually(session).Should(gbytes.Say(`Stopping a restore can leave the system in bad state. Are you sure you want to cancel\? \[yes/no\]`))
						})

						By("buffering the logs", func() {
							Expect(string(session.Out.Contents())).To(HaveSuffix(fmt.Sprintf("[yes/no]\n")))
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
							time.Sleep(time.Millisecond * 100) // without this sleep, the following assertion won't ever fail, even if the session does exit
							Expect(session.Exited).NotTo(BeClosed(), "bbr process terminated in response to signal")
						})

						By("outputting a helpful message", func() {
							Eventually(session).Should(gbytes.Say(`Stopping a restore can leave the system in bad state. Are you sure you want to cancel\? \[yes/no\]`))
						})

						By("buffering the logs", func() {
							Expect(string(session.Out.Contents())).To(HaveSuffix(fmt.Sprintf("[yes/no]\n")))
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
					Expect(session.Err.Contents()).To(ContainSubstring("dear lord"))
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
					stackTrace, err := ioutil.ReadFile(logFilePath)
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
					Expect(session.Err.Contents()).To(ContainSubstring(
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
					stackTrace, err := ioutil.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})

				By("not deleting the artifact", func() {
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
				})
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
						JobID:   "fake-uuid",
					},
					{
						IPs:     []string{"10.0.0.10"},
						JobName: "redis-server",
						JobID:   "fake-uuid",
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

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-server-0-redis.tar", backupContents)
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
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
					Expect(string(session.Err.Contents())).To(ContainSubstring("job locking dependency graph is cyclic"))
				})
			})
		})
	})

	Context("when deployment has named artifacts, with a default artifact", func() {
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
						JobID:   "fake-uuid",
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

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tar", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", createTarWithContents(map[string]string{}))
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
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
				Expect(instance1.FileExists("/var/vcap/store/redis-server" +
					"/redis-backup")).To(BeTrue())
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
			})
		})
	})

	Context("when deployment has named artifacts, without a default artifact", func() {
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
						JobName: "redis-restore-node",
						JobID:   "fake-uuid",
					},
					{
						IPs:     []string{"10.0.0.2"},
						JobName: "redis-backup-node",
						JobID:   "fake-uuid",
					}}),
				DownloadManifest(deploymentName, manifest),
				SetupSSH(deploymentName, "redis-restore-node", "fake-uuid", 0, instance1),
				SetupSSH(deploymentName, "redis-backup-node", "fake-uuid", 0, instance2),
				CleanupSSH(deploymentName, "redis-restore-node"),
				CleanupSSH(deploymentName, "redis-backup-node"))...)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
restore_name: foo
"`)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY/* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)
			instance2.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh
set -u
echo "dosent matter"`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-backup-node
  index: 0
custom_artifacts:
- name: foo
  checksums:
    ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tar", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-backup-node-0.tar", createTarWithContents(map[string]string{}))
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
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
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
			})

			By("running the restore script on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
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

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tar", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-backup-node-0.tar", createTarWithContents(map[string]string{}))
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
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
		})

		It("writes the stack trace", func() {
			files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
			Expect(err).NotTo(HaveOccurred())
			logFilePath := files[0]
			_, err = os.Stat(logFilePath)
			Expect(os.IsNotExist(err)).To(BeFalse())
			stackTrace, err := ioutil.ReadFile(logFilePath)
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

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
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
				Expect(session.Err.Contents()).To(ContainSubstring("cleanup err"))
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
