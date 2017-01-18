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
	"os"
)

type Instance struct {
	dockerID string
}

var testclusterTimeout = 5 * time.Second

func NewInstance() *Instance {
	contents := dockerRunAndWaitForSuccess("run", "--publish", "22", "--detach", "cloudfoundrylondon/backup-and-restore-node-with-ssh")

	dockerID := strings.TrimSpace(contents)

	return &Instance{
		dockerID: dockerID,
	}
}

func (i *Instance) Address() string {
	return strings.TrimSpace(strings.Replace(dockerRunAndWaitForSuccess("port", i.dockerID, "22"), "0.0.0.0", i.dockerHostIp(),-1))
}

func (i *Instance) dockerHostIp() string {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost==""{
		return "0.0.0.0"
	} else {
		return strings.Split(dockerHost, ":")[0]
	}
}
func (i *Instance) CreateUser(username, key string) {
	dockerRunAndWaitForSuccess("exec", i.dockerID, "/bin/create_user_with_key", username, key)
}

func (i *Instance) CreateFiles(files ...string) {
	for _, fileName := range files {
		dockerRunAndWaitForSuccess("exec", i.dockerID, "mkdir", "-p", filepath.Dir(fileName))
		dockerRunAndWaitForSuccess("exec", i.dockerID, "touch", fileName)
		dockerRunAndWaitForSuccess("exec", i.dockerID, "chmod", "+x", fileName)
	}
}

func (i *Instance) CreateDir(path string) {
	dockerRunAndWaitForSuccess("exec", i.dockerID, "mkdir", "-p", path)
}

func (i *Instance) RunInBackground(command string) {
	dockerRunAndWaitForSuccess("exec", "-d", i.dockerID, command)
}
func (i *Instance) CreateScript(file, contents string) {
	dockerRunAndWaitForSuccess("exec", i.dockerID, "mkdir", "-p", filepath.Dir(file))
	dockerRunAndWaitForSuccess("exec", i.dockerID, "sh", "-c", fmt.Sprintf(`echo '%s' > %s`, contents, file))
	dockerRunAndWaitForSuccess("exec", i.dockerID, "chmod", "+x", file)
}

func (i *Instance) FileExists(path string) bool {
	session := dockerRun("exec", i.dockerID, "ls", path)
	Eventually(session).Should(gexec.Exit())
	return session.ExitCode() == 0
}

func (i *Instance) GetFileContents(path string) string {
	return dockerRunAndWaitForSuccess("exec", i.dockerID, "cat", path)
}

func (i *Instance) DieInBackground() {
	go func() {
		defer GinkgoRecover()
		i.die()
	}()
}

func (i *Instance) die() {
	if i != nil {
		dockerRunAndWaitForSuccess("kill", i.dockerID)
	}
}

func dockerRun(args ...string) *gexec.Session {
	cmd := exec.Command("docker", args...)
	fmt.Fprintf(GinkgoWriter, "Starting docker run %v\n", args)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

func dockerRunAndWaitForSuccess(args ...string) string {
	startTime := time.Now()
	session := dockerRun(args...)
	Eventually(session, testclusterTimeout).Should(gexec.Exit(0))
	fmt.Fprintf(GinkgoWriter, "Completed docker run in %v, cmd: %v\n", time.Now().Sub(startTime), args)
	return string(session.Out.Contents())
}
