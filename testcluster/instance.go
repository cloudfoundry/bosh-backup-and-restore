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

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"github.com/onsi/gomega/gexec"
)

type Instance struct {
	dockerID string
}

const timeout = 40 * time.Second

func PullDockerImage() {
	startTime := time.Now()
	args := []string{"pull", "cryogenics/jammy-stemcell-oci:latest"}
	session := dockerRun(args...)
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
	fmt.Fprintf(GinkgoWriter, "Completed docker run in %v, cmd: %v\n", time.Now().Sub(startTime), args) //nolint:errcheck,staticcheck
}

func NewInstance() *Instance {
	contents := dockerRunAndWaitForSuccess("run", "--publish", "22", "--detach", "cryogenics/jammy-stemcell-oci:latest", "bash", "-c", `#!/bin/bash

	mkdir -p /var/run/sshd
	echo "%sudo ALL = (ALL) NOPASSWD: ALL" >> /etc/sudoers
	mkdir -p /var/vcap/store
	mkdir -p /var/vcap/jobs
	ssh-keygen -A
	/usr/sbin/sshd  # this will be restarted down the road, so we need a placekeep process to keep the container alive
	while true; do sleep 1; done
	`,
	)

	dockerID := strings.TrimSpace(contents)
	found := false
	fixturePath := `../fixtures/create_user_with_key`
	for !found {
		_, err := os.Stat(fixturePath)
		found = err == nil
		if found {
			break
		}
		fixturePath = fmt.Sprintf("../%s", fixturePath)
	}
	dockerRunAndWaitForSuccess("cp", fixturePath, fmt.Sprintf("%s:/bin/create_user_with_key", dockerID))
	dockerRunAndWaitForSuccess(`exec`, dockerID, "chmod", "+x", "/bin/create_user_with_key")
	dockerRunAndWaitForSuccess("exec", dockerID, "bash", "-c", `until nc -q 0 localhost 22; do sleep 1; done`)
	return &Instance{
		dockerID: dockerID,
	}
}

func NewInstanceWithKeepAlive(aliveInterval int) *Instance {
	instance := NewInstance()

	dockerRunAndWaitForSuccess("exec", instance.dockerID, "sed", "-i", fmt.Sprintf("s/^ClientAliveInterval .*/ClientAliveInterval %d/g", aliveInterval), "/etc/ssh/sshd_config")
	dockerRunAndWaitForSuccess("exec", "--detach", instance.dockerID, "pkill", "sshd")
	dockerRunAndWaitForSuccess("exec", "--detach", instance.dockerID, "/usr/sbin/sshd")
	dockerRunAndWaitForSuccess("exec", instance.dockerID, "bash", "-c", `until nc -q 0 localhost 22; do sleep 1; done`)

	return instance
}

func (mockInstance *Instance) Address() string {
	localMapsForContainerPort22 := dockerRunAndWaitForSuccess("port", mockInstance.dockerID, "22")
	localIPv4MapForContainerPort22Slice := strings.Split(localMapsForContainerPort22, "\n")
	Expect(localIPv4MapForContainerPort22Slice).NotTo(BeNil())
	Expect(len(localIPv4MapForContainerPort22Slice)).NotTo(BeZero())
	localIPv4MapForContainerPort22 := localIPv4MapForContainerPort22Slice[0]
	return strings.TrimSpace(strings.Replace(localIPv4MapForContainerPort22, "0.0.0.0", mockInstance.dockerHostIp(), -1)) //nolint:staticcheck
}

func (mockInstance *Instance) IP() string {
	return mockInstance.dockerHostIp()
}

func (mockInstance *Instance) dockerHostIp() string {
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
func (mockInstance *Instance) CreateUser(username, key string) {
	dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "/bin/create_user_with_key", username, key)
}

func (mockInstance *Instance) CreateExecutableFiles(files ...string) {
	for _, fileName := range files {
		dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "mkdir", "-p", filepath.Dir(fileName))
		dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "touch", fileName)
		dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "chmod", "+x", fileName)
	}
}

func (mockInstance *Instance) CreateDir(path string) {
	dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "mkdir", "-p", path)
}

func (mockInstance *Instance) RunInBackground(command string) {
	dockerRunAndWaitForSuccess("exec", "-d", mockInstance.dockerID, command)
}

func (mockInstance *Instance) Run(command ...string) string {
	args := append([]string{"exec", mockInstance.dockerID}, command...)
	return dockerRunAndWaitForSuccess(args...)
}

func (mockInstance *Instance) CreateScript(file, contents string) {
	dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "mkdir", "-p", filepath.Dir(file))
	dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "sh", "-c", fmt.Sprintf(`echo '%s' > %s`, contents, file))
	dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "chmod", "+x", file)
}

func (mockInstance *Instance) FileExists(path string) bool {
	session := dockerRun("exec", mockInstance.dockerID, "ls", path)
	Eventually(session, 1*time.Minute).Should(gexec.Exit())
	return session.ExitCode() == 0
}

func (mockInstance *Instance) GetFileContents(path string) string {
	return dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "cat", path)
}

func (mockInstance *Instance) GetCreatedTime(path string) string {
	return dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "/usr/bin/stat", "-c", "%y", path)
}

var waitGroup sync.WaitGroup

func (mockInstance *Instance) DieInBackground() {
	if mockInstance != nil {
		waitGroup.Add(1)
		go func() {
			defer GinkgoRecover()
			defer waitGroup.Done()
			Eventually(dockerRun("kill", mockInstance.dockerID), timeout).Should(gexec.Exit())
			Eventually(dockerRun("rm", mockInstance.dockerID), timeout).Should(gexec.Exit())
		}()
	}
}

func (mockInstance *Instance) HostPublicKey() string {
	dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "bash", "-c", "until cat /etc/ssh/ssh_host_rsa_key.pub; do sleep 1; done")
	return dockerRunAndWaitForSuccess("exec", mockInstance.dockerID, "perl", "-p", "-e", "s/\n/ /", "/etc/ssh/ssh_host_rsa_key.pub")
}

func WaitForContainersToDie() {
	waitGroup.Wait()
}

func dockerRun(args ...string) *gexec.Session {
	cmd := exec.Command("docker", args...)
	fmt.Fprintf(GinkgoWriter, "Starting docker run %v\n", args) //nolint:errcheck
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

func dockerRunAndWaitForSuccess(args ...string) string {
	startTime := time.Now()
	session := dockerRun(args...)
	Eventually(session, timeout).Should(gexec.Exit(0))
	fmt.Fprintf(GinkgoWriter, "Completed docker run in %v, cmd: %v\n", time.Now().Sub(startTime), args) //nolint:errcheck,staticcheck
	return string(session.Out.Contents())
}
