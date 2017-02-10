package integration

import (
	"io/ioutil"
	"os"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/pivotal-cf/pcf-backup-and-restore/testcluster"

	"archive/tar"
	"bytes"
	"compress/gzip"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restore", func() {
	var director *mockhttp.Server
	var restoreWorkspace string

	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		restoreWorkspace, err = ioutil.TempDir(".", "restore-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(restoreWorkspace)).To(Succeed())
		director.VerifyMocks()
	})

	Context("when deployment is not present", func() {
		var session *gexec.Session
		deploymentName := "my-new-deployment"

		BeforeEach(func() {
			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances: []`))

			director.VerifyAndMock(mockbosh.VMsForDeployment(deploymentName).NotFound())
			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", "my-new-deployment",
				"restore")

		})
		It("fails", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})
		It("prints an error", func() {
			Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
		})
	})

	Context("when artifact is not present", func() {
		var session *gexec.Session

		BeforeEach(func() {
			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", "my-new-deployment",
				"restore")

		})
		It("fails", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})
		It("prints an error", func() {
			Expect(string(session.Err.Contents())).To(ContainSubstring("no such file or directory"))
		})
	})

	Context("when the backup artifact is corrupted", func() {
		var session *gexec.Session
		var deploymentName string
		BeforeEach(func() {
			deploymentName = "my-new-deployment"

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- instance_name: redis-dedicated-node
  instance_index: 0
  checksums:
    redis-backup: this-is-not-a-checksum-this-is-only-a-tribute
`))

			backupContents, err := ioutil.ReadFile("../fixtures/backup.tgz")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0.tgz", backupContents)
			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", "my-new-deployment",
				"restore")

		})
		It("fails", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})
		It("prints an error", func() {
			Expect(string(session.Err.Contents())).To(ContainSubstring("Backup artifact is corrupted"))
		})
	})

	Context("when deployment has a single instance", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: "redis-dedicated-node",
					JobID:   "fake-uuid",
				}}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSH(deploymentName, "redis-dedicated-node"))...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/p-restore", `#!/usr/bin/env sh
set -u
cp -r $ARTIFACT_DIRECTORY* /var/vcap/store/redis-server`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- instance_name: redis-dedicated-node
  instance_index: 0
  checksums:
    ./redis/redis-backup: e1b615ac53a1ef01cf2d4021941f9d56db451fd8`))

			backupContents, err := ioutil.ReadFile("../fixtures/backup.tgz")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0.tgz", backupContents)
		})

		JustBeforeEach(func() {
			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
			instance1.DieInBackground()
		})

		It("does not fail", func() {
			Expect(session.ExitCode()).To(Equal(0))
		})

		It("Cleans up the archive file on the remote", func() {
			Expect(instance1.FileExists("/var/vcap/store/backup/redis-backup")).To(BeFalse())
		})

		It("Runs the restore script on the remote", func() {
			Expect(instance1.FileExists("" +
				"/redis-backup"))
		})

		Context("when restore fails", func() {
			BeforeEach(func() {
				instance1.CreateScript("/var/vcap/jobs/redis/bin/p-restore", `#!/usr/bin/env sh
	>&2 echo "dear lord"; exit 1`)
			})

			It("fails", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			It("returns the failure", func() {
				Expect(session.Err.Contents()).To(ContainSubstring("dear lord"))
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
			director.VerifyAndMock(AppendBuilders(VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
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
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				SetupSSH(deploymentName, "redis-server", "fake-uuid", 0, instance2),
				CleanupSSH(deploymentName, "redis-dedicated-node"),
				CleanupSSH(deploymentName, "redis-server"))...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/p-restore", `#!/usr/bin/env sh
set -u
cp -r $ARTIFACT_DIRECTORY* /var/vcap/store/redis-server`)
			instance2.CreateScript("/var/vcap/jobs/redis/bin/p-restore", `#!/usr/bin/env sh
set -u
cp -r $ARTIFACT_DIRECTORY* /var/vcap/store/redis-server`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- instance_name: redis-dedicated-node
  instance_index: 0
  checksums:
    ./redis/redis-backup: e1b615ac53a1ef01cf2d4021941f9d56db451fd8
- instance_name: redis-server
  instance_index: 0
  checksums:
    ./redis/redis-backup: e1b615ac53a1ef01cf2d4021941f9d56db451fd8`))

			backupContents, err := ioutil.ReadFile("../fixtures/backup.tgz")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0.tgz", backupContents)
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-server-0.tgz", backupContents)
		})

		JustBeforeEach(func() {
			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
			instance1.DieInBackground()
		})

		It("does not fail", func() {
			Expect(session.ExitCode()).To(Equal(0))
		})

		It("Cleans up the archive file on the remote", func() {
			Expect(instance1.FileExists("/var/vcap/store/backup/redis-backup")).To(BeFalse())
			Expect(instance2.FileExists("/var/vcap/store/backup/redis-backup")).To(BeFalse())
		})

		It("Runs the restore script on the remote", func() {
			Expect(instance1.FileExists("" +
				"/redis-backup"))
			Expect(instance2.FileExists("" +
				"/redis-backup"))
		})
	})

	Context("when deployment has named artifacts, with a default artifact", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: "redis-dedicated-node",
					JobID:   "fake-uuid",
				}}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSH(deploymentName, "redis-dedicated-node"))...)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/p-metadata", `#!/usr/bin/env sh
echo "---
restore_name: foo
"`)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/p-restore", `#!/usr/bin/env sh
set -u
cp -r $ARTIFACT_DIRECTORY* /var/vcap/store/redis-server`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- instance_name: redis-dedicated-node
  instance_index: 0
  checksums: {}
blobs:
- blob_name: foo
  checksums:
    ./redis/redis-backup: e1b615ac53a1ef01cf2d4021941f9d56db451fd8`))

			backupContents, err := ioutil.ReadFile("../fixtures/backup.tgz")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tgz", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0.tgz", gzipContents(createTarWithContents(map[string]string{})))
		})

		JustBeforeEach(func() {
			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
			instance1.DieInBackground()
		})

		It("does not fail", func() {
			Expect(session.ExitCode()).To(Equal(0))
		})

		It("Cleans up the archive file on the remote", func() {
			Expect(instance1.FileExists("/var/vcap/store/backup")).To(BeFalse())
		})

		It("Runs the restore script on the remote", func() {
			Expect(instance1.FileExists("" +
				"/redis-backup"))
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
			director.VerifyAndMock(AppendBuilders(VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
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
				SetupSSH(deploymentName, "redis-restore-node", "fake-uuid", 0, instance1),
				SetupSSH(deploymentName, "redis-backup-node", "fake-uuid", 0, instance2),
				CleanupSSH(deploymentName, "redis-restore-node"),
				CleanupSSH(deploymentName, "redis-backup-node"))...)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/p-metadata", `#!/usr/bin/env sh
echo "---
restore_name: foo
"`)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/p-restore", `#!/usr/bin/env sh
set -u
cp -r $ARTIFACT_DIRECTORY* /var/vcap/store/redis-server`)
			instance2.CreateScript("/var/vcap/jobs/redis/bin/p-backup", `#!/usr/bin/env sh
set -u
echo "dosent matter"`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- instance_name: redis-backup-node
  instance_index: 0
  checksums: {}
blobs:
- blob_name: foo
  checksums:
    ./redis/redis-backup: e1b615ac53a1ef01cf2d4021941f9d56db451fd8`))

			backupContents, err := ioutil.ReadFile("../fixtures/backup.tgz")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tgz", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-backup-node-0.tgz", gzipContents(createTarWithContents(map[string]string{})))
		})

		JustBeforeEach(func() {
			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
			instance1.DieInBackground()
			instance2.DieInBackground()
		})

		It("does not fail", func() {
			Expect(session.ExitCode()).To(Equal(0))
		})

		It("Cleans up the archive file on the remote", func() {
			Expect(instance1.FileExists("/var/vcap/store/backup")).To(BeFalse())
		})

		It("Runs the restore script on the remote", func() {
			Expect(instance1.FileExists("" +
				"/redis-backup"))
		})
	})

	Context("when the backup with named artifacts on disk is corrupted", func() {
		var session *gexec.Session
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "my-new-deployment"

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- instance_name: redis-backup-node
  instance_index: 0
  checksums: {}
blobs:
- blob_name: foo
  checksums:
    ./redis/redis-backup: this-is-damn-wrong`))
			director.VerifyAndMock()

			backupContents, err := ioutil.ReadFile("../fixtures/backup.tgz")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tgz", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-backup-node-0.tgz", gzipContents(createTarWithContents(map[string]string{})))
		})

		JustBeforeEach(func() {
			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore")
		})

		It("fails", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})

		It("does not connect to the BOSH director", func() {
			director.VerifyMocks()
		})
	})

	Context("the cleanup fails", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: "redis-dedicated-node",
				}}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSHFails(deploymentName, "redis-dedicated-node", "cleanup err"))...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/p-restore", `#!/usr/bin/env sh
cp -r $ARTIFACT_DIRECTORY* /var/vcap/store/`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- instance_name: redis-dedicated-node
  instance_index: 0
  checksums:
    ./redis/redis-backup: e1b615ac53a1ef01cf2d4021941f9d56db451fd8`))

			backupContents, err := ioutil.ReadFile("../fixtures/backup.tgz")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0.tgz", backupContents)
		})

		JustBeforeEach(func() {
			session = runBinary(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
			instance1.DieInBackground()
		})

		It("fails", func() {
			Expect(session.ExitCode()).To(Equal(2))
		})

		It("Cleans up the archive file on the remote", func() {
			Expect(instance1.FileExists("/var/vcap/store/backup/redis-backup")).To(BeFalse())
		})

		It("Runs the restore script on the remote", func() {
			Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup"))
		})

		It("returns the failure", func() {
			Expect(session.Err.Contents()).To(ContainSubstring("cleanup err"))
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

func gzipContents(contents []byte) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	gzipStream := gzip.NewWriter(bytesBuffer)
	gzipStream.Write(contents)

	Expect(gzipStream.Close()).NotTo(HaveOccurred())
	return bytesBuffer.Bytes()
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
