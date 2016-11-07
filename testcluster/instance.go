package testcluster

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type Instance struct {
	dockerID string
}

var testclusterTimeout = 5 * time.Second

func NewInstance() *Instance {
	contents := dockerRun("run", "--publish", "22", "--detach", "cloudfoundrylondon/backup-and-restore-node-with-ssh")

	dockerID := strings.TrimSpace(contents)

	return &Instance{
		dockerID: dockerID,
	}
}

func (i *Instance) Address() string {
	return strings.TrimSpace(dockerRun("port", i.dockerID, "22"))
}
func (i *Instance) CreateUser(username, key string) {
	dockerRun("exec", i.dockerID, "/bin/create_user_with_key", username, key)
}

func (i *Instance) FilesExist(files ...string) {
	for _, fileName := range files {
		dockerRun("exec", i.dockerID, "mkdir", "-p", filepath.Dir(fileName))
		dockerRun("exec", i.dockerID, "touch", fileName)
		dockerRun("exec", i.dockerID, "chmod", "+x", fileName)
	}
}

func (i *Instance) RunInBackground(command string) {
	dockerRun("exec", "-d", i.dockerID, command)
}
func (i *Instance) ScriptExist(file, contents string) {
	dockerRun("exec", i.dockerID, "mkdir", "-p", filepath.Dir(file))
	dockerRun("exec", i.dockerID, "sh", "-c", fmt.Sprintf(`echo '%s' > %s`, contents, file))
	dockerRun("exec", i.dockerID, "chmod", "+x", file)
}

func (i *Instance) Die() {
	dockerRun("kill", i.dockerID)
}

func dockerRun(args ...string) string {
	cmd := exec.Command("docker", args...)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, testclusterTimeout).Should(gexec.Exit(0))
	return string(session.Out.Contents())
}
