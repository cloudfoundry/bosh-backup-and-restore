package integration

import (
	"fmt"
	"os/exec"

	"time"

	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"gopkg.in/yaml.v2"
	"crypto/sha256"
)

type Binary struct {
	path       string
	runTimeout time.Duration
}

func NewBinary(path string) Binary {
	return Binary{path: path, runTimeout: 99999 * time.Hour}
}

func (b Binary) Run(cwd string, env []string, params ...string) *gexec.Session {
	command := exec.Command(b.path, params...)
	command.Env = env
	command.Dir = cwd
	fmt.Fprintf(GinkgoWriter, "Running command: %v %v in %s with env %v\n", b.path, params, cwd, env)
	fmt.Fprintf(GinkgoWriter, "Command output start\n")
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, b.runTimeout).Should(gexec.Exit())
	fmt.Fprintf(GinkgoWriter, "Command output end\n")
	fmt.Fprintf(GinkgoWriter, "Exited with %d\n", session.ExitCode())

	return session
}

type instanceMetadata struct {
	InstanceName  string                   `yaml:"name"`
	InstanceIndex string                   `yaml:"index"`
	Artifacts     []customArtifactMetadata `yaml:"artifacts"`
}

type customArtifactMetadata struct {
	Name      string            `yaml:"name"`
	Checksums map[string]string `yaml:"checksums"`
}

type backupActivityMetadata struct {
	StartTime  string `yaml:"start_time"`
	FinishTime string `yaml:"finish_time"`
}

type metadata struct {
	InstancesMetadata       []instanceMetadata       `yaml:"instances"`
	CustomArtifactsMetadata []customArtifactMetadata `yaml:"custom_artifacts,omitempty"`
	BackupActivityMetadata  backupActivityMetadata   `yaml:"backup_activity"`
}

func ParseMetadata(filePath string) metadata {
	metadataContents := metadata{}
	contents, _ := ioutil.ReadFile(filePath)
	yaml.Unmarshal(contents, &metadataContents)
	return metadataContents
}

func ShaFor(contents string) string {
	shasum := sha256.New()
	shasum.Write([]byte(contents))
	return fmt.Sprintf("%x", shasum.Sum(nil))
}