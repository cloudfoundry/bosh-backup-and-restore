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
	"github.com/pivotal-cf/bosh-backup-and-restore/testcluster"

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
	var instance1 *testcluster.Instance

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

	Context("When there is a deployment which has one instance", func() {
		var metadataFile string
		var redisNodeArtifactFile string

		singleInstanceResponse := func(instanceGroupName string) []mockbosh.VMsOutput {
			return []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: instanceGroupName,
				},
			}
		}

		Context("when there is a plausible backup script", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				By("creating a dummy backup script")
				instance1.CreateScript("/var/vcap/jobs/redis/bin/b-backup", `#!/usr/bin/env sh

set -u

printf "backupcontent1" > $ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $ARTIFACT_DIRECTORY/backupdump2
`)

				mockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					DownloadManifest(deploymentName, "this is a totally valid yaml"),
					CleanupSSH(deploymentName, "redis-dedicated-node"))

				metadataFile = path.Join(backupWorkspace, deploymentName, "/metadata")
				redisNodeArtifactFile = path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0.tgz")
			})

			Context("and there are no pre-backup scripts", func() {

				It("exits zero", func() {
					Expect(session.ExitCode()).To(BeZero())
				})

				It("downloads the manifest", func() {
					Expect(path.Join(backupWorkspace, deploymentName, "manifest.yml")).To(BeARegularFile())
					Expect(ioutil.ReadFile(path.Join(backupWorkspace, deploymentName, "manifest.yml"))).To(MatchYAML("this is a totally valid yaml"))
				})

				It("creates a backup directory which contains a backup artifact", func() {
					Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
					Expect(redisNodeArtifactFile).To(BeARegularFile())
				})

				It("the backup artifact contains the backup files from the instance", func() {
					Expect(filesInTar(redisNodeArtifactFile)).To(ConsistOf("backupdump1", "backupdump2"))
					Expect(contentsInTar(redisNodeArtifactFile, "backupdump1")).To(Equal("backupcontent1"))
					Expect(contentsInTar(redisNodeArtifactFile, "backupdump2")).To(Equal("backupcontent2"))
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
					Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Running pre-checks for backup of %s...", deploymentName)))
					Expect(session.Out).To(gbytes.Say("INFO - Scripts found:"))
					Expect(session.Out).To(gbytes.Say("INFO - redis-dedicated-node/fake-uuid/redis/b-backup"))
					Expect(session.Out).To(gbytes.Say(fmt.Sprintf("INFO - Starting backup of %s...", deploymentName)))
					Expect(session.Out).To(gbytes.Say("INFO - Running pre-backup scripts..."))
					Expect(session.Out).To(gbytes.Say("INFO - Done."))
					Expect(session.Out).To(gbytes.Say("INFO - Running backup scripts..."))
					Expect(session.Out).To(gbytes.Say("INFO - Backing up redis on redis-dedicated-node/fake-uuid..."))
					Expect(session.Out).To(gbytes.Say("INFO - Done."))
					Expect(session.Out).To(gbytes.Say("INFO - Running post-backup scripts..."))
					Expect(session.Out).To(gbytes.Say("INFO - Done."))
					Expect(session.Out).To(gbytes.Say("INFO - Copying backup -- [^-]*-- from redis-dedicated-node/fake-uuid..."))
					Expect(session.Out).To(gbytes.Say("INFO - Finished copying backup -- from redis-dedicated-node/fake-uuid..."))
					Expect(session.Out).To(gbytes.Say("INFO - Starting validity checks"))
					Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./redis/backupdump[12]`))
					Expect(session.Out).To(gbytes.Say(`DEBUG - Calculating shasum for local file ./redis/backupdump[12]`))
					Expect(session.Out).To(gbytes.Say("DEBUG - Calculating shasum for remote files"))
					Expect(session.Out).To(gbytes.Say("DEBUG - Comparing shasums"))
					Expect(session.Out).To(gbytes.Say("INFO - Finished validity checks"))

				})

				It("cleans up backup artifacts from remote", func() {
					Expect(instance1.FileExists("/var/vcap/store/backup")).To(BeFalse())
				})
			})

			Context("when there is a b-metadata script which produces yaml containing the custom backup_name", func() {
				var redisCustomArtifactFile string
				var redisDefaultArtifactFile string

				BeforeEach(func() {
					instance1.CreateScript("/var/vcap/jobs/redis/bin/b-metadata", `#!/usr/bin/env sh
	touch /tmp/b-metadata-output
echo "---
backup_name: foo_redis
"`)
					redisCustomArtifactFile = path.Join(backupWorkspace, deploymentName, "/foo_redis.tgz")
					redisDefaultArtifactFile = path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0.tgz")
				})

				It("runs the b-metadata scripts", func() {
					Expect(instance1.FileExists("/tmp/b-metadata-output")).To(BeTrue())
				})

				It("creates a custom backup artifact", func() {
					Expect(filesInTar(redisCustomArtifactFile)).To(ConsistOf("backupdump1", "backupdump2"))
					Expect(contentsInTar(redisCustomArtifactFile, "backupdump1")).To(Equal("backupcontent1"))
					Expect(contentsInTar(redisCustomArtifactFile, "backupdump2")).To(Equal("backupcontent2"))
				})

				It("does not create an artifact with the default name", func() {
					Expect(redisDefaultArtifactFile).NotTo(BeARegularFile())
				})

				It("the metadata records the artifact as a blob instead of an instance", func() {
					Expect(metadataFile).To(BeARegularFile())
					Expect(ioutil.ReadFile(metadataFile)).To(MatchYAML(fmt.Sprintf(`instances: []
blobs:
- blob_name: foo_redis
  checksums:
    ./backupdump1: %s
    ./backupdump2: %s
`, shaFor("backupcontent1"), shaFor("backupcontent2"))))
				})
			})

			Context("when the b-pre-backup-lock script is present", func() {
				BeforeEach(func() {
					instance1.CreateScript("/var/vcap/jobs/redis/bin/b-pre-backup-lock", `#!/usr/bin/env sh
touch /tmp/pre-backup-lock-output
`)
					instance1.CreateScript("/var/vcap/jobs/redis-broker/bin/b-pre-backup-lock", ``)
				})

				It("runs the b-pre-backup-lock scripts", func() {
					Expect(instance1.FileExists("/tmp/pre-backup-lock-output")).To(BeTrue())
				})

				It("logs that it is locking the instance, and lists the scripts", func() {
					assertOutput(session, []string{
						`Locking redis on redis-dedicated-node/fake-uuid for backup`,
						"> /var/vcap/jobs/redis/bin/b-pre-backup-lock",
						"> /var/vcap/jobs/redis-broker/bin/b-pre-backup-lock",
					})
				})
			})

			Context("when the b-pre-backup-lock script fails", func() {
				BeforeEach(func() {
					instance1.CreateScript("/var/vcap/jobs/redis/bin/b-pre-backup-lock", `#!/usr/bin/env sh
echo 'ultra-bar'
(>&2 echo 'ultra-baz')
touch /tmp/pre-backup-lock-output
exit 1
`)
					instance1.CreateScript("/var/vcap/jobs/redis-broker/bin/b-pre-backup-lock", ``)
					instance1.CreateScript("/var/vcap/jobs/redis/bin/b-post-backup-unlock", `#!/usr/bin/env sh
touch /tmp/post-backup-unlock-output
`)
				})

				It("runs the b-pre-backup-lock scripts", func() {
					Expect(instance1.FileExists("/tmp/pre-backup-lock-output")).To(BeTrue())
				})

				It("exits with the correct error code", func() {
					Expect(session.ExitCode()).To(Equal(4))
				})

				It("logs the error", func() {
					Expect(session.Err.Contents()).To(ContainSubstring("pre backup lock script for job redis failed on redis-dedicated-node/fake-uuid."))
				})

				It("logs stdout", func() {
					Expect(session.Err.Contents()).To(ContainSubstring("Stdout: ultra-bar"))
				})

				It("logs stderr", func() {
					Expect(session.Err.Contents()).To(ContainSubstring("Stderr: ultra-baz"))
				})

				It("also runs the b-post-backup-unlock scripts", func() {
					Expect(instance1.FileExists("/tmp/post-backup-unlock-output")).To(BeTrue())
				})
			})

			Context("when backup file has owner only permissions of different user", func() {
				BeforeEach(func() {
					instance1.CreateScript("/var/vcap/jobs/redis/bin/b-backup", `#!/usr/bin/env sh

set -u

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
					instance1.CreateScript("/var/vcap/jobs/redis/bin/b-post-backup-unlock", `#!/usr/bin/env sh
echo "Unlocking release"`)
				})

				It("prints unlock progress to the screen", func() {
					assertOutput(session, []string{
						"Running unlock on redis-dedicated-node/fake-uuid",
						"Done.",
					})
				})

				Context("when the post backup unlock script fails", func() {
					BeforeEach(func() {
						instance1.CreateScript("/var/vcap/jobs/redis/bin/b-post-backup-unlock", `#!/usr/bin/env sh
echo 'ultra-bar'
(>&2 echo 'ultra-baz')
exit 1`)
					})

					It("exits with the correct error code", func() {
						Expect(session).To(gexec.Exit(8))
					})

					It("prints stdout", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("Stdout: ultra-bar"))
					})

					It("prints stderr", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("Stderr: ultra-baz"))
					})

					It("prints an error", func() {
						Expect(session.Err.Contents()).To(ContainSubstring("unlock script for job redis failed on redis-dedicated-node/fake-uuid."))
					})
				})
			})
		})

		Context("when a deployment can't be backed up", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				mockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					ManifestIsNotDownloaded(),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

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

		Context("when the instance backup script fails", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				mockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					DownloadManifest(deploymentName, "this is a totally valid yaml"),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/b-backup", "echo 'ultra-bar'; (>&2 echo 'ultra-baz'); exit 1",
				)
			})

			It("returns exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})
		})

		Context("when both the instance backup script and cleanup fail", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				mockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					DownloadManifest(deploymentName, "this is a totally valid yaml"),
					CleanupSSHFails(deploymentName, "redis-dedicated-node", "ultra-foo"),
				)

				instance1.CreateScript(
					"/var/vcap/jobs/redis/bin/b-backup", "(>&2 echo 'ultra-baz'); exit 1",
				)
			})

			It("returns a exit code 17 (16 + 1)", func() {
				Expect(session.ExitCode()).To(Equal(17))
			})

			It("prints an error", func() {
				assertErrorOutput(session, []string{
					"backup script for job redis failed on redis-dedicated-node/fake-uuid.",
					"ultra-baz",
					"ultra-foo",
				})
			})
		})

		Context("when backup succeeds but cleanup fails", func() {
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				mockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					DownloadManifest(deploymentName, "this is a totally valid yaml"),
					CleanupSSHFails(deploymentName, "redis-dedicated-node", "Can't do it mate"),
				)

				instance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/b-backup",
				)
			})

			It("returns the correct error code", func() {
				Expect(session.ExitCode()).To(Equal(16))
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

		Context("when running the b-metadata script does not give valid yml", func() {
			AfterEach(func() {
				instance1.DieInBackground()
			})
			BeforeEach(func() {
				instance1 = testcluster.NewInstance()
				instance1.CreateScript("/var/vcap/jobs/redis/bin/b-metadata", `#!/usr/bin/env sh
touch /tmp/b-metadata-output
echo "not valid yaml
"`)

				mockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					ManifestIsNotDownloaded(),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)
			})

			It("runs the b-metadata scripts", func() {
				Expect(instance1.FileExists("/tmp/b-metadata-output")).To(BeTrue())
			})

			It("exits with the correct error code", func() {
				Expect(session).To(gexec.Exit(1))
			})

		})

		Context("when the artifact exists locally", func() {
			BeforeEach(func() {
				director.VerifyAndMock(mockbosh.Info().WithAuthTypeBasic())
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

	Context("When there is a deployment which has two instances", func() {
		twoInstancesResponse := func(firstInstanceGroupName, secondInstanceGroupName string) []mockbosh.VMsOutput {

			return []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: firstInstanceGroupName,
				},
				{
					IPs:     []string{"10.0.0.2"},
					JobName: secondInstanceGroupName,
				},
			}
		}

		Context("one backupable", func() {
			var backupableInstance, nonBackupableInstance *testcluster.Instance

			BeforeEach(func() {
				deploymentName = "my-bigger-deployment"
				backupableInstance = testcluster.NewInstance()
				nonBackupableInstance = testcluster.NewInstance()
				mockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, twoInstancesResponse("redis-dedicated-node", "redis-broker")),
					append(SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, backupableInstance),
						SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, nonBackupableInstance)...),
					DownloadManifest(deploymentName, "not being asserted"),
					append(CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-broker")...),
				)
				backupableInstance.CreateFiles(
					"/var/vcap/jobs/redis/bin/b-backup",
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

		Context("both backupable", func() {
			var backupableInstance1, backupableInstance2 *testcluster.Instance

			BeforeEach(func() {
				deploymentName = "my-two-instance-deployment"
				backupableInstance1 = testcluster.NewInstance()
				backupableInstance2 = testcluster.NewInstance()
				mockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, twoInstancesResponse("redis-dedicated-node", "redis-broker")),
					append(SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, backupableInstance1),
						SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, backupableInstance2)...),
					DownloadManifest(deploymentName, "not being asserted"),
					append(CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-broker")...),
				)

				backupableInstance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/b-backup",
				)

				backupableInstance2.CreateFiles(
					"/var/vcap/jobs/redis/bin/b-backup",
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
				assertOutput(session, []string{
					fmt.Sprintf("Starting backup of %s...", deploymentName),
					"Backing up redis on redis-dedicated-node/fake-uuid...",
					"Backing up redis on redis-broker/fake-uuid-2...",
					"Done.",
					"Copying backup --",
					"from redis-dedicated-node/fake-uuid...",
					"from redis-broker/fake-uuid-2...",
					"Done.",
					fmt.Sprintf("Backup created of %s on", deploymentName),
				})
			})

		})

		Context("both specify the same backup name in their metadata", func() {
			var backupableInstance1, backupableInstance2 *testcluster.Instance

			BeforeEach(func() {
				deploymentName = "my-two-instance-deployment"
				backupableInstance1 = testcluster.NewInstance()
				backupableInstance2 = testcluster.NewInstance()
				mockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, twoInstancesResponse("redis-dedicated-node", "redis-broker")),
					append(SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, backupableInstance1),
						SetupSSH(deploymentName, "redis-broker", "fake-uuid-2", 0, backupableInstance2)...),
					ManifestIsNotDownloaded(),
					append(CleanupSSH(deploymentName, "redis-dedicated-node"),
						CleanupSSH(deploymentName, "redis-broker")...),
				)

				backupableInstance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/b-backup",
				)

				backupableInstance2.CreateFiles(
					"/var/vcap/jobs/redis/bin/b-backup",
				)

				backupableInstance1.CreateScript("/var/vcap/jobs/redis/bin/b-metadata", `#!/usr/bin/env sh
echo "---
backup_name: duplicate_name
"`)
				backupableInstance2.CreateScript("/var/vcap/jobs/redis/bin/b-metadata", `#!/usr/bin/env sh
echo "---
backup_name: duplicate_name
"`)
			})

			AfterEach(func() {
				backupableInstance1.DieInBackground()
				backupableInstance2.DieInBackground()
			})

			It("files with the name are not created", func() {
				Expect(path.Join(backupWorkspace, deploymentName, "/duplicate_name.tgz")).NotTo(BeARegularFile())
			})

			It("refuses to perform backup", func() {
				Expect(session.Err.Contents()).To(ContainSubstring(
					"Multiple jobs in deployment 'my-two-instance-deployment' specified the same backup name",
				))
			})

			It("returns exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})
		})
	})

	Context("When deployment does not exist", func() {
		BeforeEach(func() {
			deploymentName = "my-non-existent-deployment"
			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeBasic(),
				mockbosh.VMsForDeployment(deploymentName).NotFound(),
			)
		})

		It("returns exit code 1", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})

		It("prints an error", func() {
			Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
		})

	})
})

func getTarReader(path string) *tar.Reader {
	reader, err := os.Open(path)
	Expect(err).NotTo(HaveOccurred())
	defer reader.Close()
	archive, err := gzip.NewReader(reader)
	Expect(err).NotTo(HaveOccurred())
	tarReader := tar.NewReader(archive)
	return tarReader
}

func filesInTar(path string) []string {
	tarReader := getTarReader(path)

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
	tarReader := getTarReader(tarFile)

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

func shaFor(contents string) string {
	shasum := sha1.New()
	shasum.Write([]byte(contents))
	return fmt.Sprintf("%x", shasum.Sum(nil))
}

func assertOutput(session *gexec.Session, strings []string) {
	for _, str := range strings {
		Expect(string(session.Out.Contents())).To(ContainSubstring(str))
	}
}

func assertErrorOutput(session *gexec.Session, strings []string) {
	for _, str := range strings {
		Expect(string(session.Err.Contents())).To(ContainSubstring(str))
	}
}
