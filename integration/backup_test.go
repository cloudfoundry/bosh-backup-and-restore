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

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup", func() {
	var director *mockhttp.Server
	var backupWorkspace string

	BeforeEach(func() {
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

	Context("with deployment, with one instance present", func() {
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "my-little-deployment"
			instance1 = testcluster.NewInstance()
			director.VerifyAndMock(
				mockbosh.VMsForDeployment(deploymentName).RedirectsToTask(14),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(14).RespondsWithVMsOutput([]string{}),
				mockbosh.TaskOutput(14).RespondsWithVMsOutput([]mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
					},
				}),
				mockbosh.StartSSHSession(deploymentName).SetSSHResponseCallback(func(username, key string) {
					instance1.CreateUser(username, key)
				}).RedirectsToTask(15),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(15).RespondsWith("{}"),
				mockbosh.TaskOutput(15).RespondsWith(fmt.Sprintf(`[{"status":"success",
				"ip":"%s",
				"host_public_key":"not-relevant",
				"index":0}]`, instance1.Address())),
				mockbosh.CleanupSSHSession(deploymentName).RedirectsToTask(16),
				mockbosh.Task(16).RespondsWithTaskContainingState(mockbosh.TaskDone),
			)
		})

		AfterEach(func() {
			go instance1.Die()
		})

		Context("when the backup is successful", func() {
			var session *gexec.Session
			var backupArtifactFile string
			var metadataFile string
			var outputFile string

			BeforeEach(func() {
				instance1.ScriptExist("/var/vcap/jobs/redis/bin/backup", `#!/usr/bin/env sh
printf "backupcontent1" > /var/vcap/store/backup/backupdump1
printf "backupcontent2" > /var/vcap/store/backup/backupdump2
`)
				session = runBinary(
					backupWorkspace,
					[]string{"BOSH_PASSWORD=admin"},
					"--ca-cert", sslCertPath,
					"--username", "admin",
					"--target", director.URL,
					"--deployment", deploymentName,
					"--debug",
					"backup",
				)
				backupArtifactFile = path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0.tgz")
				metadataFile = path.Join(backupWorkspace, deploymentName, "/metadata")
				outputFile = path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0.tgz")
			})

			It("exits zero", func() {
				Expect(session.ExitCode()).To(BeZero())
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
				shasumOfTar := shaForFile(backupArtifactFile)
				Expect(ioutil.ReadFile(metadataFile)).To(MatchYAML(`instances:
- instance_name: redis-dedicated-node
  instance_id: "0"
  checksum: ` + shasumOfTar))
			})
		})

		Context("if a deployment can't be backed up", func() {
			var session *gexec.Session
			BeforeEach(func() {
				session = runBinary(
					backupWorkspace,
					[]string{"BOSH_PASSWORD=admin"},
					"--ca-cert", sslCertPath,
					"--username", "admin",
					"--target", director.URL,
					"--deployment", deploymentName,
					"--debug",
					"backup",
				)
				instance1.FilesExist(
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
	})

	Context("with deployment, with two instances (one backupable)", func() {
		var backupableInstance *testcluster.Instance
		var nonBackupableInstance *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "my-bigger-deployment"
			backupableInstance = testcluster.NewInstance()
			nonBackupableInstance = testcluster.NewInstance()
			director.VerifyAndMock(
				mockbosh.VMsForDeployment(deploymentName).RedirectsToTask(14),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(14).RespondsWithVMsOutput([]string{}),
				mockbosh.TaskOutput(14).RespondsWithVMsOutput([]mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
					},
					{
						IPs:     []string{"10.0.0.2"},
						JobName: "redis-broker",
					},
				}),
				mockbosh.StartSSHSession(deploymentName).ForInstanceGroup("redis-dedicated-node").
					SetSSHResponseCallback(func(username, key string) {
						backupableInstance.CreateUser(username, key)
					}).RedirectsToTask(15),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(15).RespondsWith("{}"),
				mockbosh.TaskOutput(15).RespondsWith(fmt.Sprintf(`[{"status":"success",
				"ip":"%s",
				"host_public_key":"not-relevant",
				"index":0}]`, backupableInstance.Address())),

				mockbosh.StartSSHSession(deploymentName).ForInstanceGroup("redis-broker").
					SetSSHResponseCallback(func(username, key string) {
						nonBackupableInstance.CreateUser(username, key)
					}).RedirectsToTask(19),
				mockbosh.Task(19).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(19).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(19).RespondsWith("{}"),
				mockbosh.TaskOutput(19).RespondsWith(fmt.Sprintf(`[{"status":"success",
				"ip":"%s",
				"host_public_key":"not-relevant",
				"index":0}]`, nonBackupableInstance.Address())),

				mockbosh.CleanupSSHSession(deploymentName).ForInstanceGroup("redis-dedicated-node").RedirectsToTask(16),
				mockbosh.Task(16).RespondsWithTaskContainingState(mockbosh.TaskDone),

				mockbosh.CleanupSSHSession(deploymentName).ForInstanceGroup("redis-broker").RedirectsToTask(20),
				mockbosh.Task(20).RespondsWithTaskContainingState(mockbosh.TaskDone),
			)
		})

		AfterEach(func() {
			go backupableInstance.Die()
			go nonBackupableInstance.Die()
		})

		It("backs up deployment successfully", func() {
			backupableInstance.FilesExist(
				"/var/vcap/jobs/redis/bin/backup",
			)

			session := runBinary(backupWorkspace, []string{"BOSH_PASSWORD=admin"}, "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", deploymentName, "--debug", "backup")

			Expect(session.ExitCode()).To(BeZero())
			Expect(path.Join(backupWorkspace, deploymentName)).To(BeADirectory())
			Expect(path.Join(backupWorkspace, deploymentName, "/redis-dedicated-node-0.tgz")).To(BeARegularFile())
			Expect(path.Join(backupWorkspace, deploymentName, "/redis-broker-0.tgz")).ToNot(BeAnExistingFile())
		})

	})

	Context("when deployment does not exist", func() {
		var session *gexec.Session
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "my-non-existent-deployment"
			director.VerifyAndMock(mockbosh.VMsForDeployment(deploymentName).NotFound())
			session = runBinary(
				backupWorkspace,
				[]string{"BOSH_PASSWORD=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", deploymentName,
				"backup",
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
	shasum := sha1.New()
	shasum.Write(contents)
	return fmt.Sprintf("%x", shasum.Sum(nil))
}
