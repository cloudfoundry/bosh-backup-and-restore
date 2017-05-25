package testcluster

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"net"
	"net/url"
	"os"

	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type Instance struct {
	dockerID string
}

const timeout = 10 * time.Second

func NewInstance() *Instance {
	contents := dockerRunAndWaitForSuccess("run", "--publish", "22", "--detach", "cloudfoundrylondon/backup-and-restore-node-with-ssh")

	dockerID := strings.TrimSpace(contents)

	return &Instance{
		dockerID: dockerID,
	}
}

func NewInstanceWithKeepAlive(aliveInterval int) *Instance {
	contents := dockerRunAndWaitForSuccess("run", "--publish", "22", "--detach", "cloudfoundrylondon/backup-and-restore-node-with-ssh", "tail", "-f", "/dev/null")

	dockerID := strings.TrimSpace(contents)

	instance := &Instance{
		dockerID: dockerID,
	}
	dockerRunAndWaitForSuccess("exec", instance.dockerID, "sed", "-i", fmt.Sprintf("s/^ClientAliveInterval .*/ClientAliveInterval %d/g", aliveInterval), "/etc/ssh/sshd_config")
	dockerRunAndWaitForSuccess("exec", "--detach", instance.dockerID, "/usr/sbin/sshd")

	return instance
}

func (i *Instance) Address() string {
	return strings.TrimSpace(strings.Replace(dockerRunAndWaitForSuccess("port", i.dockerID, "22"), "0.0.0.0", i.dockerHostIp(), -1))
}

func (i *Instance) dockerHostIp() string {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		return "0.0.0.0"
	} else {
		uri, err := url.Parse(dockerHost)
		Expect(err).NotTo(HaveOccurred())
		host, _, err := net.SplitHostPort(uri.Host)
		Expect(err).NotTo(HaveOccurred())
		return host
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

func (i *Instance) Run(command ...string) string {
	args := append([]string{"exec", i.dockerID}, command...)
	return dockerRunAndWaitForSuccess(args...)
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

var waitGroup sync.WaitGroup

func (i *Instance) DieInBackground() {
	if i != nil {
		waitGroup.Add(1)
		go func() {
			defer GinkgoRecover()
			defer waitGroup.Done()
			Eventually(dockerRun("kill", i.dockerID), timeout).Should(gexec.Exit())
			Eventually(dockerRun("rm", i.dockerID), timeout).Should(gexec.Exit())
		}()
	}
}

func WaitForContainersToDie() {
	waitGroup.Wait()
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
	Eventually(session, timeout).Should(gexec.Exit(0))
	fmt.Fprintf(GinkgoWriter, "Completed docker run in %v, cmd: %v\n", time.Now().Sub(startTime), args)
	return string(session.Out.Contents())
}
