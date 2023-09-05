package director

import (
	"fmt"
	"os"
	"strings"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"

	"github.com/onsi/gomega/gexec"

	"os/exec"

	"path"

	"path/filepath"

	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Restore", func() {
	var restoreWorkspace string
	var session *gexec.Session
	var directorAddress, directorIP string
	var artifactName string
	var waitForRestoreToFinish bool
	var stdin io.WriteCloser

	BeforeEach(func() {
		waitForRestoreToFinish = true
		var err error
		restoreWorkspace, err = os.MkdirTemp(".", "restore-workspace-")
		Expect(err).NotTo(HaveOccurred())
		artifactName = "director-backup-integration"

		command := exec.Command("cp", "-r", "../../fixtures/director-backup-integration", path.Join(restoreWorkspace, artifactName))
		cpFiles, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(cpFiles).Should(gexec.Exit())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(restoreWorkspace)).To(Succeed())
	})

	JustBeforeEach(func() {
		env := []string{"BOSH_CLIENT_SECRET=admin", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))}

		params := []string{
			"director",
			"--host", directorAddress,
			"--username", "foobar",
			"--private-key-path", pathToPrivateKeyFile,
			"--debug",
			"restore",
			"--artifact-path", artifactName,
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

	Context("When there is a director instance", func() {
		var directorInstance *testcluster.Instance

		BeforeEach(func() {
			directorInstance = testcluster.NewInstance()
			directorInstance.CreateUser("foobar", readFile(pathToPublicKeyFile))
			directorAddress = directorInstance.Address()
			directorIP = directorInstance.IP()
		})

		AfterEach(func() {
			directorInstance.DieInBackground()
		})

		Context("and there are restore scripts", func() {
			BeforeEach(func() {
				directorInstance.CreateExecutableFiles("/var/vcap/jobs/bosh/bin/bbr/restore")
				directorInstance.CreateExecutableFiles("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock")
			})

			Context("and the restore script succeeds", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/restore", `#!/usr/bin/env sh
set -u

mkdir -p /var/vcap/store/bosh/
cat $BBR_ARTIFACT_DIRECTORY/backup > /var/vcap/store/bosh/restored_file
`)
				})

				It("successfully restores to the director", func() {
					By("exiting zero", func() {
						Expect(session.ExitCode()).To(BeZero())
					})

					By("logging the steps it takes", func() {
						Expect(session.Out).To(gbytes.Say("INFO - Starting restore of"))
						Expect(session.Out).To(gbytes.Say("INFO - Validating backup artifact for"))
						Expect(session.Out).To(gbytes.Say("INFO - Looking for scripts"))
						Expect(session.Out).To(gbytes.Say("INFO - Copying backup -- 4.0K uncompressed -- for job bosh on bosh/0..."))
						Expect(session.Out).To(gbytes.Say("INFO - Finished copying backup for job bosh on bosh/0."))
						Expect(session.Out).To(gbytes.Say("INFO - Completed restore of"))
					})

					By("running the restore script successfully", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
						Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
					})

					By("running the restore script successfully", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
						Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
					})

					By("cleaning up backup artifacts from the remote", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
					})
				})

				Context("and the bbr process receives SIGINT while restore", func() {
					BeforeEach(func() {
						waitForRestoreToFinish = false

						By("creating a restore script that takes a while")
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/restore", `#!/usr/bin/env sh

set -u

sleep 5

mkdir -p /var/vcap/store/bosh/
cat $BBR_ARTIFACT_DIRECTORY/backup > /var/vcap/store/bosh/restored_file
				`)
					})

					Context("and the user decides to cancel the restore", func() {
						It("terminates", func() {
							session.Interrupt()

							By("not terminating", func() {
								Consistently(session.Exited).ShouldNot(BeClosed(), "bbr exited without user confirmation")
							})

							By("outputting a helpful message", func() {
								Eventually(session).Should(gbytes.Say(`Stopping a restore can leave the system in bad state. Are you sure you want to cancel\? \[yes/no\]`))
							})

							stdin.Write([]byte("yes\n"))

							By("waiting for the restore to finish successfully", func() {
								Eventually(session, 10).Should(gexec.Exit(1))
							})

							By("outputting a warning about cleanup", func() {
								Eventually(session).Should(gbytes.Say("It is recommended that you run `bbr restore-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."))
							})

							By("not completing the restore", func() {
								Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeFalse())
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

							stdin.Write([]byte("no\n"))

							By("waiting for the restore to finish successfully", func() {
								Eventually(session, 10).Should(gexec.Exit(0))
							})

							By("still completing the restore", func() {
								Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
								Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
							})
						})
					})
				})

				Context("there is a pre-restore-lock script which succeeds", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/pre-restore-lock-script-was-run
`)
					})
					It("runs the pre-restore-lock script successfully", func() {
						By("exiting zero", func() {
							Expect(session.ExitCode()).To(BeZero())
						})

						By("running the pre-restore-lock script successfully", func() {
							Expect(directorInstance.FileExists("/tmp/pre-restore-lock-script-was-run")).To(BeTrue())
						})

						By("running the restore script successfully", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
							Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
						})

						By("cleaning up backup artifacts from the remote", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
						})
					})
				})

				Context("and there is a pre-restore-lock script which fails", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
echo "pre-restore-lock errored!"
exit 1
`)
					})
					It("fails the command", func() {
						By("exiting non-zero", func() {
							Expect(session.ExitCode()).NotTo(BeZero())
							Expect(session.Out).To(gbytes.Say("pre-restore-lock errored"))
						})

						By("not running the restore script", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeFalse())
						})

						By("cleaning up backup artifacts from the remote", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
						})

					})

					Context("there is a post-restore-unlock script which succeeds", func() {
						BeforeEach(func() {
							directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run
`)
						})

						It("fails the command", func() {
							By("exiting non-zero", func() {
								Expect(session.ExitCode()).NotTo(BeZero())
								Expect(session.Out).To(gbytes.Say("pre-restore-lock errored"))
							})

							By("not running the restore script", func() {
								Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeFalse())
							})

							By("cleaning up backup artifacts from the remote", func() {
								Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
							})
							By("running the post-restore-unlock script successfully", func() {
								Expect(directorInstance.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
							})
						})
					})
				})

				Context("there is a post-restore-unlock script which succeeds", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run
`)
					})
					It("runs the post-restore-unlock script successfully", func() {
						By("exiting zero", func() {
							Expect(session.ExitCode()).To(BeZero())
						})

						By("running the restore script successfully", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
							Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
						})

						By("running the post-restore-unlock script successfully", func() {
							Expect(directorInstance.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
						})

						By("cleaning up backup artifacts from the remote", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
						})
					})
				})

				Context("and there is a post-restore-unlock script which fails", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
echo "post-restore-unlock errored!"
exit 1
`)
					})
					It("fails the command", func() {
						By("exiting non-zero", func() {
							Expect(session.ExitCode()).NotTo(BeZero())
						})

						By("running the restore script successfully", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
							Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
						})

						By("cleaning up backup artifacts from the remote", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
						})

						By("error is displayed", func() {
							Expect(session.Out).To(gbytes.Say("post-restore-unlock errored"))
						})
					})

				})
			})

			Context("but the restore script fails", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/restore", "echo 'NOPE!'; exit 1")
				})

				It("fails to restore the director", func() {
					By("returning exit code 1", func() {
						Expect(session.ExitCode()).To(Equal(1))
						Expect(session.Out).To(gbytes.Say("NOPE!"))
					})
				})

				Context("there is a post-restore-unlock script which succeeds", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run
`)
					})
					It("runs the post-restore-unlock script successfully", func() {
						Expect(directorInstance.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
					})
				})

				Context("and there is a post-restore-unlock script which fails", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
echo "post-restore-unlock errored!"
exit 1
`)
					})
					It("fails the command", func() {
						By("exiting non-zero", func() {
							Expect(session.ExitCode()).NotTo(BeZero())
						})

						By("error is displayed", func() {
							Expect(session.Out).To(gbytes.Say("NOPE!"))
							Expect(session.Out).To(gbytes.Say("post-restore-unlock errored"))
						})
					})

				})
			})

			Context("but the artifact directory already exists", func() {
				BeforeEach(func() {
					directorInstance.CreateDir("/var/vcap/store/bbr-backup")
				})

				It("fails to restore the director", func() {
					By("exiting non-zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					By("printing a log message saying the director instance cannot be backed up", func() {
						Expect(session.Err).To(gbytes.Say("Directory /var/vcap/store/bbr-backup already exists on instance bosh/0"))
					})

					By("not deleting the existing artifact directory", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
					})
				})
			})

			Context("with ordering on pre-restore-lock specified", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
restore_should_be_locked_before:
- job_name: postgres
  release: bosh
"`)
					directorInstance.CreateScript(
						"/var/vcap/jobs/postgres/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/postgres-pre-restore-lock-called
exit 0`)

					directorInstance.CreateScript(
						"/var/vcap/jobs/bosh/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/bosh-pre-restore-lock-called
exit 0`)
					directorInstance.CreateScript(
						"/var/vcap/jobs/postgres/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/postgres-post-restore-unlock-called
exit 0`)

					directorInstance.CreateScript(
						"/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/bosh-post-restore-unlock-called
exit 0`)
				})

				It("locks in the specified order", func() {
					Expect(directorInstance.FileExists("/tmp/postgres-pre-restore-lock-called")).To(BeTrue())
					postgresJobLockTime := directorInstance.GetCreatedTime("/tmp/postgres-pre-restore-lock-called")

					Expect(directorInstance.FileExists("/tmp/bosh-pre-restore-lock-called")).To(BeTrue())
					boshJobLockTime := directorInstance.GetCreatedTime("/tmp/bosh-pre-restore-lock-called")

					Expect(session.Out).To(gbytes.Say("Detected order: bosh should be locked before postgres during restore"))

					Expect(boshJobLockTime < postgresJobLockTime).To(BeTrue(), fmt.Sprintf(
						"'bosh' locked at %s, which is after the 'postgres' locked (%s)",
						strings.TrimSuffix(boshJobLockTime, "\n"),
						strings.TrimSuffix(postgresJobLockTime, "\n")))
				})

				It("unlocks in the right order", func() {
					By("unlocking the postgres job before unlocking the bosh job")
					postgresJobUnlockTime := directorInstance.GetCreatedTime("/tmp/postgres-post-restore-unlock-called")
					boshJobUnlockTime := directorInstance.GetCreatedTime("/tmp/bosh-post-restore-unlock-called")

					Expect(postgresJobUnlockTime < boshJobUnlockTime).To(BeTrue(), fmt.Sprintf(
						"'bosh' job unlocked at %s, which is before the 'postgres' job unlocked (%s)",
						strings.TrimSuffix(boshJobUnlockTime, "\n"),
						strings.TrimSuffix(postgresJobUnlockTime, "\n")))
				})
			})

			Context("but the pre-restore-lock ordering is cyclic", func() {
				BeforeEach(func() {
					directorInstance.CreateScript(
						"/var/vcap/jobs/bosh/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/bosh-pre-restore-lock-called
exit 0`)
					directorInstance.CreateScript(
						"/var/vcap/jobs/postgres/bin/bbr/pre-restore-lock", `#!/usr/bin/env sh
touch /tmp/postgres-writer-pre-restore-lock-called
exit 0`)
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
restore_should_be_locked_before:
- job_name: postgres
  release: bosh
"`)
					directorInstance.CreateScript("/var/vcap/jobs/postgres/bin/bbr/metadata",
						`#!/usr/bin/env sh
echo "---
restore_should_be_locked_before:
- job_name: bosh
  release: bosh
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

		Context("but there are no restore scripts", func() {
			BeforeEach(func() {
				directorInstance.CreateExecutableFiles("/var/vcap/jobs/bosh/bin/bbr/backup")
				directorInstance.CreateExecutableFiles("/var/vcap/jobs/bosh/bin/bbr/not-a-restore-script")
			})

			It("fails to restore the director", func() {
				By("returning exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("printing an error", func() {
					Expect(session.Err).To(gbytes.Say(fmt.Sprintf("Deployment '%s' has no restore scripts", directorIP)))
				})

				By("saving the stack trace into a file", func() {
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
	})

	Context("When the director does not resolve", func() {
		BeforeEach(func() {
			directorAddress = "does-not-resolve"
		})

		It("fails to restore the director", func() {
			By("returning exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(session.Err).To(SatisfyAny(
					gbytes.Say("no such host"),
					gbytes.Say("server misbehaving"),
					gbytes.Say("Temporary failure in name resolution")))
			})
		})
	})
})
