package deployment

import (
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

var _ = Describe("Restore cleanup", func() {
	var cleanupWorkspace string
	var director *mockhttp.Server

	var session *gexec.Session
	var instance *testcluster.Instance
	var deploymentName string
	var err error
	manifest := `---
instance_groups:
- name: redis-dedicated-node
  instances: 1
  jobs:
  - name: redis
    release: redis
`

	BeforeEach(func() {
		cleanupWorkspace, err = ioutil.TempDir(".", "cleanup-workspace-")

		instance = testcluster.NewInstance()

		deploymentName = "my-new-deployment"
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		director.VerifyAndMock(AppendBuilders(
			InfoWithBasicAuth(),
			VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: "redis-dedicated-node",
					JobID:   "fake-uuid",
					ID:      "fake-uuid",
					Index:   newIndex(0),
				}}),
			DownloadManifest(deploymentName, manifest),
			SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance),
			CleanupSSH(deploymentName, "redis-dedicated-node"),
		)...)

		instance.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", ``)
		instance.CreateDir("/var/vcap/store/bbr-backup")
	})

	JustBeforeEach(func() {
		session = binary.Run(
			cleanupWorkspace,
			[]string{"BOSH_CLIENT_SECRET=admin"},
			"deployment",
			"--ca-cert", sslCertPath,
			"--username", "admin",
			"--debug",
			"--target", director.URL,
			"--deployment", deploymentName,
			"restore-cleanup",
		)
	})

	AfterEach(func() {
		instance.DieInBackground()
		Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
	})

	It("successfully cleans up a deployment after a failed restore", func() {
		Eventually(session.ExitCode()).Should(Equal(0))
		Expect(instance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
	})
})
