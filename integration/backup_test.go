package integration

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/pivotal-cf/pcf-backup-and-restore/testcluster"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup", func() {
	var director *mockhttp.Server
	var backupWorkspace string

	AfterEach(func() {
		director.VerifyMocks()
	})
	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
	})

	Context("with deployment, with one instance present", func() {
		var instance1 *testcluster.Instance

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			director.VerifyAndMock(
				mockbosh.VMsForDeployment("my-new-deployment").RedirectsToTask(14),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(14).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(14).RespondsWithVMsOutput([]string{}),
				mockbosh.TaskOutput(14).RespondsWithVMsOutput([]mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
					},
				}),
				mockbosh.StartSSHSession("my-new-deployment").SetSSHResponseCallback(func(username, key string) {
					instance1.CreateUser(username, key)
				}).RedirectsToTask(15),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.Task(15).RespondsWithTaskContainingState(mockbosh.TaskDone),
				mockbosh.TaskEvent(15).RespondsWith("{}"),
				mockbosh.TaskOutput(15).RespondsWith(fmt.Sprintf(`[{"status":"success",
				"ip":"%s",
				"host_public_key":"not-relevant",
				"index":0}]`, instance1.Address())),
				mockbosh.CleanupSSHSession("my-new-deployment").RedirectsToTask(16),
				mockbosh.Task(16).RespondsWithTaskContainingState(mockbosh.TaskDone),
			)
		})

		AfterEach(func() {
			go instance1.Die()
		})

		It("backs up deployment successfully", func() {
			instance1.ScriptExist("/var/vcap/jobs/redis/bin/backup", `#!/usr/bin/env sh
printf "backupcontent1" > /var/vcap/store/backup/backupdump1
printf "backupcontent2" > /var/vcap/store/backup/backupdump2
`)

			session := runBinary(backupWorkspace, []string{"BOSH_PASSWORD=admin"}, "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "--debug", "backup")

			Expect(session.ExitCode()).To(BeZero())
			Expect(path.Join(backupWorkspace, "my-new-deployment")).To(BeADirectory())
			Expect(path.Join(backupWorkspace, "my-new-deployment/redis-dedicated-node-0.tgz")).To(BeARegularFile())
			outputFile := path.Join(backupWorkspace, "my-new-deployment/redis-dedicated-node-0.tgz")
			Expect(filesInTar(outputFile)).To(ConsistOf("backupdump1", "backupdump2"))
			Expect(contentsInTar(outputFile, "backupdump1")).To(Equal("backupcontent1"))
			Expect(contentsInTar(outputFile, "backupdump2")).To(Equal("backupcontent2"))

			Expect(path.Join(backupWorkspace, "my-new-deployment/metadata")).To(BeARegularFile())
		})

		It("errors if a deployment cant be backuped", func() {
			instance1.FilesExist(
				"/var/vcap/jobs/redis/bin/ctl",
			)

			session := runBinary(backupWorkspace, []string{"BOSH_PASSWORD=admin"}, "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "--debug", "backup")
			Expect(session.ExitCode()).NotTo(BeZero())
			Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment 'my-new-deployment' has no backup scripts"))
			Expect(path.Join(backupWorkspace, "my-new-deployment")).NotTo(BeADirectory())
		})
	})

	Context("with deployment, with two instances (one backupable)", func() {
		var backupableInstance *testcluster.Instance
		var nonBackupableInstance *testcluster.Instance

		BeforeEach(func() {
			backupableInstance = testcluster.NewInstance()
			nonBackupableInstance = testcluster.NewInstance()
			director.VerifyAndMock(
				mockbosh.VMsForDeployment("my-new-deployment").RedirectsToTask(14),
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
				mockbosh.StartSSHSession("my-new-deployment").ForInstanceGroup("redis-dedicated-node").
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

				mockbosh.StartSSHSession("my-new-deployment").ForInstanceGroup("redis-broker").
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

				mockbosh.CleanupSSHSession("my-new-deployment").ForInstanceGroup("redis-dedicated-node").RedirectsToTask(16),
				mockbosh.Task(16).RespondsWithTaskContainingState(mockbosh.TaskDone),

				mockbosh.CleanupSSHSession("my-new-deployment").ForInstanceGroup("redis-broker").RedirectsToTask(20),
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

			session := runBinary(backupWorkspace, []string{"BOSH_PASSWORD=admin"}, "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "--debug", "backup")

			Expect(session.ExitCode()).To(BeZero())
			Expect(path.Join(backupWorkspace, "my-new-deployment")).To(BeADirectory())
			Expect(path.Join(backupWorkspace, "my-new-deployment/redis-dedicated-node-0.tgz")).To(BeARegularFile())
			Expect(path.Join(backupWorkspace, "my-new-deployment/redis-broker-0.tgz")).ToNot(BeAnExistingFile())
		})

	})

	It("returns error if deployment not found", func() {
		director.VerifyAndMock(mockbosh.VMsForDeployment("my-new-deployment").NotFound())

		session := runBinary(backupWorkspace, []string{"BOSH_PASSWORD=admin"}, "--ca-cert", sslCertPath, "--username", "admin", "--target", director.URL, "--deployment", "my-new-deployment", "backup")

		Expect(session.ExitCode()).To(Equal(1))
		Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
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
