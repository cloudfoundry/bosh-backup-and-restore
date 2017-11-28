package instance

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o fakes/fake_remote_runner.go . RemoteRunner
type RemoteRunner interface {
	ConnectedUsername() string
	DirectoryExists(dir string) (bool, error)
	RemoveDirectory(dir string) error
	ArchiveAndDownload(directory string, writer io.Writer) error
	CreateDirectory(directory string) error
	ExtractAndUpload(reader io.Reader, directory string) error
	SizeOf(path string) (string, error)
	ChecksumDirectory(path string) (map[string]string, error)
	RunScript(path string) (string, error)
	RunScriptWithEnv(path string, env map[string]string) (string, error)
	FindFiles(pattern string) ([]string, error)
}

type SshRemoteRunner struct {
	logger     Logger
	connection ssh.SSHConnection
}

func NewRemoteRunner(sshConnection ssh.SSHConnection, logger Logger) SshRemoteRunner {
	return SshRemoteRunner{
		connection: sshConnection,
		logger:     logger,
	}
}

func (r SshRemoteRunner) ConnectedUsername() string {
	return r.connection.Username()
}

func (r SshRemoteRunner) DirectoryExists(dir string) (bool, error) {
	_, _, exitCode, err := r.connection.Run(fmt.Sprintf("sudo stat %s", dir))
	return exitCode == 0, err
}

func (r SshRemoteRunner) CreateDirectory(directory string) error {
	_, err := r.runOnInstance("sudo mkdir -p " + directory)
	return err
}

func (r SshRemoteRunner) RemoveDirectory(dir string) error {
	_, err := r.runOnInstance(fmt.Sprintf("sudo rm -rf %s", dir))
	return err
}

func (r SshRemoteRunner) ArchiveAndDownload(directory string, writer io.Writer) error {
	stderr, exitCode, err := r.connection.Stream(fmt.Sprintf("sudo tar -C %s -c .", directory), writer)
	return r.logAndCheckErrors([]byte{}, stderr, exitCode, err)
}

func (r SshRemoteRunner) ExtractAndUpload(reader io.Reader, directory string) error {
	stdout, stderr, exitCode, err := r.connection.StreamStdin(fmt.Sprintf("sudo sh -c 'tar -C %s -x'", directory), reader)
	return r.logAndCheckErrors(stdout, stderr, exitCode, err)
}

func (r SshRemoteRunner) SizeOf(path string) (string, error) {
	stdout, err := r.runOnInstance(fmt.Sprintf("sudo du -sh %s", path))
	if err != nil {
		return "", err
	}

	return strings.Fields(string(stdout))[0], nil
}

func (r SshRemoteRunner) ChecksumDirectory(path string) (map[string]string, error) {
	stdout, err := r.runOnInstance(fmt.Sprintf("sudo sh -c 'cd %s && find . -type f | xargs shasum -a 256'", path))
	if err != nil {
		return nil, err
	}

	return convertShasToMap(stdout), nil
}

func (r SshRemoteRunner) RunScript(path string) (string, error) {
	return r.RunScriptWithEnv(path, map[string]string{})
}

func (r SshRemoteRunner) RunScriptWithEnv(path string, env map[string]string) (string, error) {
	varNames := make([]string, 0, len(env))
	for varName := range env {
		varNames = append(varNames, varName)
	}
	sort.Strings(varNames)

	var varsList = ""
	for _, varName := range varNames {
		varsList = varsList + varName + "=" + env[varName] + " "
	}

	return r.runOnInstance("sudo " + varsList + path)
}

func (r SshRemoteRunner) FindFiles(pattern string) ([]string, error) {
	stdout, stderr, exitCode, err := r.connection.Run(fmt.Sprintf("sudo sh -c 'find %s -type f'", pattern))

	r.logger.Debug("bbr", "Stdout: %s", string(stdout))
	r.logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		return nil, err
	}

	if exitCode != 0 {
		if strings.Contains(string(stderr), "No such file or directory") {
			r.logger.Debug("", "No files found for pattern '%s'", pattern)
			return []string{}, nil
		} else {
			return nil, exitError(stderr, exitCode)
		}
	}

	output := strings.TrimSpace(string(stdout))
	return strings.Split(output, "\n"), nil
}

func (r SshRemoteRunner) runOnInstance(cmd string) (string, error) {
	stdout, stderr, exitCode, runErr := r.connection.Run(cmd)

	err := r.logAndCheckErrors(stdout, stderr, exitCode, runErr)
	if err != nil {
		return "", err
	}

	return string(stdout), nil
}

func (r SshRemoteRunner) logAndCheckErrors(stdout, stderr []byte, exitCode int, err error) error {
	r.logger.Debug("bbr", "Stdout: %s", string(stdout))
	r.logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		return err
	}

	if exitCode != 0 {
		return exitError(stderr, exitCode)
	}

	return nil
}

func exitError(stderr []byte, exitCode int) error {
	return errors.New(fmt.Sprintf("%s - exit code %d", strings.TrimSpace(string(stderr)), exitCode))
}

func convertShasToMap(shas string) map[string]string {
	mapOfSha := map[string]string{}
	shas = strings.TrimSpace(shas)
	if shas == "" {
		return mapOfSha
	}
	for _, line := range strings.Split(shas, "\n") {
		parts := strings.SplitN(line, " ", 2)
		filename := strings.TrimSpace(parts[1])
		if filename == "-" {
			continue
		}
		mapOfSha[filename] = parts[0]
	}
	return mapOfSha
}
