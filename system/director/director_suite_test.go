package director

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
	"time"
)

const bbrArtifactDirectory = "/var/vcap/store/bbr-backup"

var (
	workspaceDir              string
	commandPath               string
	directorHost              string
	directorSSHUsername       string
	directorSSHKeyPath        string
	directorBackupFixturePath string
)

func TestDirector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Director Suite")
}

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(4 * time.Minute)

	directorHost = MustHaveEnv("DIRECTOR_HOST")
	directorSSHUsername = MustHaveEnv("DIRECTOR_SSH_USERNAME")
	directorSSHKeyPath = MustHaveEnv("DIRECTOR_SSH_KEY_PATH")

	var err error
	commandPath, err = gexec.Build("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr")
	Expect(err).NotTo(HaveOccurred())

	workspaceDir, err = ioutil.TempDir("", "bbr_system_test_director")
	Expect(err).NotTo(HaveOccurred())

	directorBackupFixturePath, err = filepath.Abs("../../fixtures/director-backup")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	Expect(os.RemoveAll(workspaceDir)).To(Succeed())
})

func runBBRDirector(args ...string) *gexec.Session {
	args = append([]string{
		"director",
		"--host", directorHost,
		"--username", directorSSHUsername,
		"--private-key-path", directorSSHKeyPath,
	}, args...)
	cmd := exec.Command(commandPath, args...)
	cmd.Dir = workspaceDir

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	return session
}

func runOnDirector(command string, args ...string) *gexec.Session {
	sshArgs := []string{
		fmt.Sprintf("%s@%s", directorSSHUsername, directorHost),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-i", directorSSHKeyPath,
		"sudo", command,
	}
	sshArgs = append(sshArgs, args...)

	cmd := exec.Command("ssh", sshArgs...)

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

func mustFindBackupDir(artifactPath string) string {
	matches, err := filepath.Glob(filepath.Join(artifactPath, fmt.Sprintf("%s_*T*Z", directorHost)))
	Expect(err).NotTo(HaveOccurred())
	return matches[len(matches)-1]
}

func mustHaveFile(dir, filename string) {
	_, err := os.Stat(filepath.Join(dir, filename))
	Expect(err).ToNot(HaveOccurred())
}
