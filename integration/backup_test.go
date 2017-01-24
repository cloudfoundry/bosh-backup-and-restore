package integration

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/pivotal-cf/pcf-backup-and-restore/testcluster"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup", func() {
	var director *mockhttp.Server
	var backupWorkspace string
	var session *gexec.Session
	var deploymentName string

	BeforeEach(func() {
		deploymentName = "my-little-deployment"
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
		director.VerifyMocks()
	})

	JustBeforeEach(func() {
		session = runBinary(
			backupWorkspace,
			[]string{"BOSH_CLIENT_SECRET=admin"},
			"--ca-cert", sslCertPath,
			"--username", "admin",
			"--target", director.URL,
			"--deployment", deploymentName,
			"--debug",
			"backup",
		)
	})

	Context("with deployment, with one instance present", func() {
		var instance1 *testcluster.Instance

		Context("when the backup is successful", func() {
			var backupArtifactFile string
			var metadataFile string
			var outputFile string

			AfterEach(func() {
				instance1.DieInBackground()
			})

			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				director.VerifyAndMock(AppendBuilders(
					VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
						},
					}),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					DownloadManifest(deploymentName, "this is a totally valid yaml"),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)...)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/p-backup", `#!/usr/bin/env sh
printf "backupcontent1" > $ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $ARTIFACT_DIRECTORY/backupdump2
`)

				backupArtifactFile = path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0.tgz")
				metadataFile = path.Join(backupWorkspace, deploymentName, "/metadata")
				outputFile = path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0.tgz")
			})

			Context("when the p-pre-backup-lock script is present", func() {
				BeforeEach(func() {
					instance1.CreateScript("/var/vcap/jobs/redis/bin/p-pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/pre-backup-lock-output
`)
					instance1.CreateScript("/var/vcap/jobs/redis-broker/bin/p-pre-backup-lock", ``)
				})

				It("runs the p-pre-backup-lock scripts", func() {
					Expect(instance1.FileExists("/tmp/pre-backup-lock-output")).To(BeTrue())
				})

				It("logs that it is locking the instance, and lists the scripts", func() {
					Expect(session.Out.Contents()).Should(ContainSubstring(`Locking redis-dedicated-node/fake-uuid for backup`))
					Expect(session.Out.Contents()).Should(ContainSubstring("> /var/vcap/jobs/redis/bin/p-pre-backup-lock"))
					Expect(session.Out.Contents()).Should(ContainSubstring("> /var/vcap/jobs/redis-broker/bin/p-pre-backup-lock"))
				})
			})

			Context("when the p-pre-backup-lock script fails", func() {
				BeforeEach(func() {
					instance1.CreateScript("/var/vcap/jobs/redis/bin/p-pre-backup-lock", `#!/usr/bin/env sh
echo 'ultra-bar'
(>&2 echo 'ultra-baz')
touch /tmp/pre-backup-lock-output
exit 1
`)
					instance1.CreateScript("/var/vcap/jobs/redis-broker/bin/p-pre-backup-lock", ``)
					instance1.CreateScript("/var/vcap/jobs/redis/bin/p-post-backup-unlock", `#!/usr/bin/env sh
touch /tmp/post-backup-unlock-output
`)
				})

				It("runs the p-pre-backup-lock scripts", func() {
					Expect(instance1.FileExists("/tmp/pre-backup-lock-output")).To(BeTrue())
				})

				It("logs the error", func() {
					Expect(session.Err.Contents()).To(ContainSubstring("Pre backup lock script for job redis failed on redis-dedicated-node/fake-uuid."))
				})

				It("logs stdout", func() {
					Expect(session.Err.Contents()).To(ContainSubstring("Stdout: ultra-bar"))
				})

				It("logs stderr", func() {
					Expect(session.Err.Contents()).To(ContainSubstring("Stderr: ultra-baz"))
				})

				It("also runs the p-post-backup-unlock scripts", func() {
					Expect(instance1.FileExists("/tmp/post-backup-unlock-output")).To(BeTrue())
				})
			})

			It("exits zero", func() {
				Expect(session.ExitCode()).To(BeZero())
			})

			It("downloads the manifest", func() {
				Expect(path.Join(backupWorkspace, deploymentName, "manifest.yml")).To(BeARegularFile())
				Expect(ioutil.ReadFile(path.Join(backupWorkspace, deploymentName, "manifest.yml"))).To(MatchYAML("this is a totally valid yaml"))
			})

			It("creates a backup directory which contains a backup artifact", func() {
				Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
				Expect(backupArtifactFile).To(BeARegularFile())
			})

			It("the backup artifact contains the backup files from the instance", func() {
				Expect(filesInTar(outputFile)).To(ConsistOf("backupdump1", "backupdump2"))
				Expect(contentsInTar(outputFile, "backupdump1")).To(Equal("backupcontent1"))
				Expect(contentsInTar(outputFile, "backupdump2")).To(Equal("backupcontent2"))
			})

			It("creates a metadata file", func() {
				Expect(metadataFile).To(BeARegularFile())
			})

			It("the metadata file is correct", func() {
				Expect(ioutil.ReadFile(metadataFile)).To(MatchYAML(fmt.Sprintf(`instances:
- instance_name: redis-dedicated-node
  instance_index: "0"
  checksums:
    ./redis/backupdump1: %s
    ./redis/backupdump2: %s`, shaFor("backupcontent1"), shaFor("backupcontent2"))))
			})

			It("prints the backup progress to the screen", func() {
				Eventually(session).Should(gbytes.Say("Starting backup of %s...", deploymentName))
				Eventually(session).Should(gbytes.Say("Finding instances with backup scripts..."))
				Eventually(session).Should(gbytes.Say("Done."))
				Eventually(session).Should(gbytes.Say("Backing up redis-dedicated-node/fake-uuid..."))
				Eventually(session).Should(gbytes.Say("Done."))
				Eventually(session).Should(gbytes.Say("Copying backup --"))
				Eventually(session).Should(gbytes.Say("from redis-dedicated-node/fake-uuid..."))
				Eventually(session).Should(gbytes.Say("Done."))
				Eventually(session).Should(gbytes.Say("Backup created of %s on", deploymentName))
			})

			It("cleans up backup artifacts from remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/backup")).To(BeFalse())
			})

			Context("when backup file has owner only permissions of different user", func() {
				BeforeEach(func() {
					instance1.CreateScript("/var/vcap/jobs/redis/bin/p-backup", `#!/usr/bin/env sh

dd if=/dev/urandom of=$ARTIFACT_DIRECTORY/backupdump1 bs=1KB count=1024
dd if=/dev/urandom of=$ARTIFACT_DIRECTORY/backupdump2 bs=1KB count=1024

mkdir $ARTIFACT_DIRECTORY/backupdump3
dd if=/dev/urandom of=$ARTIFACT_DIRECTORY/backupdump3/dump bs=1KB count=1024

chown vcap:vcap $ARTIFACT_DIRECTORY/backupdump3
chmod 0700 $ARTIFACT_DIRECTORY/backupdump3`)
				})

				It("exits zero", func() {
					Expect(session.ExitCode()).To(BeZero())
				})

				It("prints the artifact size with the files from the other users", func() {
					Eventually(session).Should(gbytes.Say("Copying backup -- 3.0M uncompressed -- from redis-dedicated-node/fake-uuid..."))
				})
			})

			Context("when backup deployment has a post-backup-unlock script", func() {
				BeforeEach(func() {
					instance1.CreateScript("/var/vcap/jobs/redis/bin/p-post-backup-unlock", `#!/usr/bin/env sh
echo "Unlocking release"`)
				})

				It("prints unlock progress to the screen", func() {
					Eventually(session).Should(gbytes.Say("Running unlock on redis-dedicated-node/fake-uuid"))
					Eventually(session).Should(gbytes.Say("Done."))
				})

				Context("when the post backup unlock script fails", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/p-post-backup-unlock", `#!/usr/bin/env sh
echo 'ultra-bar'
(>&2 echo 'ultra-baz')
exit 1`)
					})

					It("exits with the correct error code", func() {
						Expect(session).To(gexec.Exit(42))
					})

					It("prints stdout", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("Stdout: ultra-bar"))
					})

					It("prints stderr", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("Stderr: ultra-baz"))
					})

					It("prints an error", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("Unlock script for job redis failed on redis-dedicated-node/fake-uuid."))
					})
				})
			})
		})

		Context("if a deployment can't be backed up", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				director.VerifyAndMock(AppendBuilders(
					VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
						},
					}),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)...)

				instance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/ctl",
				)
			})

			It("returns a non-zero exit code", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("prints an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' has no backup scripts"))
			})

			It("does not create a backup on disk", func() {
				Expect(path.Join(backupWorkspace, deploymentName)).NotTo(BeADirectory())
			})
		})

		Context("instance backup script failed with an error", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				director.VerifyAndMock(AppendBuilders(
					VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
						},
					}),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					DownloadManifest(deploymentName, "this is a totally valid yaml"),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)...)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/p-backup", "echo 'ultra-bar'; (>&2 echo 'ultra-baz'); exit 1",
				)
			})

			It("returns a non-zero exit code", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

		})

		Context("instance backup script failed with an error and cleanup failed as well", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				director.VerifyAndMock(AppendBuilders(
					VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
						},
					}),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					DownloadManifest(deploymentName, "this is a totally valid yaml"),
					CleanupSSHFails(deploymentName, "redis-dedicated-node", "ultra-foo"),
				)...)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/p-backup", "(>&2 echo 'ultra-baz'); exit 1",
				)
			})

			It("returns a non-zero exit code", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			It("prints an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("Backup script for job redis failed on redis-dedicated-node/fake-uuid."))
				Expect(string(session.Err.Contents())).To(ContainSubstring("ultra-baz"))
				Expect(string(session.Err.Contents())).To(ContainSubstring("ultra-foo"))
			})
		})

		Context("if a deployment can be backed up but the cleanup fails", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				director.VerifyAndMock(AppendBuilders(
					VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
						{
							IPs:     []string{"10.0.0.1"},
							JobName: "redis-dedicated-node",
						},
					}),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					DownloadManifest(deploymentName, "this is a totally valid yaml"),
					CleanupSSHFails(deploymentName, "redis-dedicated-node", "Can't do it mate"),
				)...)

				instance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/p-backup",
				)
			})

			It("returns a partial error code", func() {
				Expect(session.ExitCode()).To(Equal(2))
			})

			It("prints an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' failed while cleaning up with error: "))
			})

			It("error output should include the failure message", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("Can't do it mate"))
			})

			It("should create a backup on disk", func() {
				Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
			})
		})

		Context("if the artifact exists locally", func() {
			BeforeEach(func() {
				deploymentName = "already-backed-up-deployment"
				err := os.Mkdir(path.Join(backupWorkspace, deploymentName), 0777)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns a non-zero exit code", func() {
				Expect(session.ExitCode()).NotTo(BeZero())
			})

			It("prints an error", func() {
				Expect(string(session.Err.Contents())).To(
					ContainSubstring(
						fmt.Sprintf("artifact %s already exists", deploymentName),
					),
				)
			})
		})
	})

	Context("with deployment, with two instances (one backupable)", func() {
		var backupableInstance, nonBackupableInstance *testcluster.Instance

		BeforeEach(func() {
			deploymentName = "my-bigger-deployment"
			backupableInstance = testcluster.NewInstance()
			nonBackupableInstance = testcluster.NewInstance()
			director.VerifyAndMock(AppendBuilders(
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
					},
					{
						IPs:     []string{"10.0.0.2"},
						JobName: "redis-broker",
					},
				}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, backupableInstance),
				SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, nonBackupableInstance),
				DownloadManifest(deploymentName, "not being asserted"),
				CleanupSSH(deploymentName, "redis-dedicated-node"),
				CleanupSSH(deploymentName, "redis-broker"),
			)...)
			backupableInstance.CreateFiles(
				"/var/vcap/jobs/redis/bin/p-backup",
			)

		})

		AfterEach(func() {
			backupableInstance.DieInBackground()
			nonBackupableInstance.DieInBackground()
		})

		It("backs up deployment successfully", func() {
			Expect(session.ExitCode()).To(BeZero())
			Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
			Expect(path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0.tgz")).To(BeARegularFile())
			Expect(path.Join(backupWorkspace, deploymentName, "/redis-broker-0.tgz")).ToNot(BeAnExistingFile())
		})
	})

	Context("with deployment, with two instances (both backupable)", func() {
		var backupableInstance1, backupableInstance2 *testcluster.Instance

		BeforeEach(func() {
			deploymentName = "my-two-instance-deployment"
			backupableInstance1 = testcluster.NewInstance()
			backupableInstance2 = testcluster.NewInstance()
			director.VerifyAndMock(AppendBuilders(
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
					},
					{
						IPs:     []string{"10.0.0.2"},
						JobName: "redis-broker",
					},
				}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, backupableInstance1),
				SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, backupableInstance2),
				DownloadManifest(deploymentName, "not being asserted"),
				CleanupSSH(deploymentName, "redis-dedicated-node"),
				CleanupSSH(deploymentName, "redis-broker"),
			)...)

			backupableInstance1.CreateFiles(
				"/var/vcap/jobs/redis/bin/p-backup",
			)

			backupableInstance2.CreateFiles(
				"/var/vcap/jobs/redis/bin/p-backup",
			)

		})

		AfterEach(func() {
			backupableInstance1.DieInBackground()
			backupableInstance2.DieInBackground()
		})

		It("backs up both instances successfully", func() {
			Expect(session.ExitCode()).To(BeZero())
			Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
			Expect(path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0.tgz")).To(BeARegularFile())
			Expect(path.Join(backupWorkspace, deploymentName, "/redis-broker-0.tgz")).To(BeARegularFile())
		})

		It("prints the backup progress to the screen", func() {
			Eventually(session).Should(gbytes.Say("Starting backup of %s...", deploymentName))
			Eventually(session).Should(gbytes.Say("Finding instances with backup scripts..."))
			Eventually(session).Should(gbytes.Say("Done."))
			Eventually(session).Should(gbytes.Say("Backing up redis-dedicated-node/fake-uuid..."))
			Eventually(session).Should(gbytes.Say("Backing up redis-broker/fake-uuid-2..."))
			Eventually(session).Should(gbytes.Say("Done."))
			Eventually(session).Should(gbytes.Say("Copying backup --"))
			Eventually(session).Should(gbytes.Say("from redis-dedicated-node/fake-uuid..."))
			Eventually(session).Should(gbytes.Say("from redis-broker/fake-uuid-2..."))
			Eventually(session).Should(gbytes.Say("Done."))
			Eventually(session).Should(gbytes.Say("Backup created of %s on", deploymentName))
		})
	})

	Context("when deployment does not exist", func() {
		BeforeEach(func() {
			deploymentName = "my-non-existent-deployment"
			director.VerifyAndMock(mockbosh.VMsForDeployment(deploymentName).NotFound())
		})

		It("returns exit code 1", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})

		It("prints an error", func() {
			Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
		})

	})
})

func filesInTar(path string) []string {
	reader, err := os.Open(path)
	Expect(err).NotTo(HaveOccurred())
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	Expect(err).NotTo(HaveOccurred())

	tarReader := tar.NewReader(archive)
	filenames := []string{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		info := header.FileInfo()
		if !info.IsDir() {
			filenames = append(filenames, info.Name())
		}
	}
	return filenames
}

func contentsInTar(tarFile, file string) string {
	reader, err := os.Open(tarFile)
	Expect(err).NotTo(HaveOccurred())
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	Expect(err).NotTo(HaveOccurred())

	tarReader := tar.NewReader(archive)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		info := header.FileInfo()
		if !info.IsDir() && info.Name() == file {
			contents, err := ioutil.ReadAll(tarReader)
			Expect(err).NotTo(HaveOccurred())
			return string(contents)
		}
	}
	Fail("File " + file + " not found in tar " + tarFile)
	return ""
}
func shaForFile(filename string) string {
	contents, err := ioutil.ReadFile(filename)
	Expect(err).NotTo(HaveOccurred())
	return shaFor(string(contents))
}

func shaFor(contents string) string {
	shasum := sha1.New()
	shasum.Write([]byte(contents))
	return fmt.Sprintf("%x", shasum.Sum(nil))
}
