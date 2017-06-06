package integration

import (
	"fmt"
	"os/exec"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var runTimeout = 99999 * time.Hour

type Binary struct {
	path string
}

func NewBinary(path string) Binary {
	return Binary{path: path}
}

func (b Binary) Run(cwd string, env []string, params ...string) *gexec.Session {
	command := exec.Command(b.path, params...)
	command.Env = env
	command.Dir = cwd
	fmt.Fprintf(GinkgoWriter, "Running command: %v %v in %s with env %v\n", b.path, params, cwd, env)
	fmt.Fprintf(GinkgoWriter, "Command output start\n")
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, runTimeout).Should(gexec.Exit())
	fmt.Fprintf(GinkgoWriter, "Command output end\n")
	fmt.Fprintf(GinkgoWriter, "Exited with %d\n", session.ExitCode())

	return session
}
