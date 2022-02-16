package ssh

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"

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
	SizeInBytes(path string) (int, error)
	ChecksumDirectory(path string) (map[string]string, error)
	RunScript(path, label string) (string, error)
	RunScriptWithEnv(path string, env map[string]string, label string, stdout io.Writer) (string, error)
	FindFiles(pattern string) ([]string, error)
	IsWindows() (bool, error)
}

type SshRemoteRunner struct {
	logger     Logger
	connection SSHConnection
}

func NewSshRemoteRunner(host, user, privateKey string, publicKeyCallback ssh.HostKeyCallback, publicKeyAlgorithm []string, logger Logger) (RemoteRunner, error) {
	connection, err := NewConnection(host, user, privateKey, publicKeyCallback, publicKeyAlgorithm, logger)
	if err != nil {
		return SshRemoteRunner{}, err
	}

	return SshRemoteRunner{
		connection: connection,
		logger:     logger,
	}, nil
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
	return r.logAndCheckErrors([]byte{}, stderr, exitCode, err, "")
}

func (r SshRemoteRunner) ExtractAndUpload(reader io.Reader, directory string) error {
	stdout, stderr, exitCode, err := r.connection.StreamStdin(fmt.Sprintf("sudo sh -c 'tar -C %s -x'", directory), reader)
	return r.logAndCheckErrors(stdout, stderr, exitCode, err, "")
}

func (r SshRemoteRunner) SizeOf(path string) (string, error) {
	stdout, err := r.runOnInstance(fmt.Sprintf("sudo du -sh %s", path))
	if err != nil {
		return "", err
	}

	return strings.Fields(string(stdout))[0], nil
}

func (r SshRemoteRunner) SizeInBytes(path string) (int, error) {
	stdout, err := r.runOnInstance(fmt.Sprintf("sudo du -s %s", path))
	if err != nil {
		return 0, err
	}
	sizeString := strings.Fields(stdout)[0]
	size, err := strconv.Atoi(sizeString)
	if err != nil {
		//untested
		return 0, fmt.Errorf("expected <%s> to be a number of bytes: failed to convert it to int", sizeString)
	}
	return size * 1024, nil
}

func (r SshRemoteRunner) ChecksumDirectory(path string) (map[string]string, error) {
	stdout, err := r.runOnInstance(fmt.Sprintf("sudo sh -c 'cd %s && find . -type f | xargs shasum -a 256'", path))
	if err != nil {
		return nil, err
	}

	return convertShasToMap(stdout), nil
}

func (r SshRemoteRunner) RunScript(path, label string) (string, error) {
	return r.RunScriptWithEnv(path, map[string]string{}, label, io.Discard)
}

func (r SshRemoteRunner) RunScriptWithEnv(path string, env map[string]string, label string, stdout io.Writer) (string, error) {
	var varsList = ""
	for varName, value := range env {
		varsList = varsList + varName + "=" + value + " "
	}

	stdoutBuffer := &bytes.Buffer{}

	stderr, exitCode, runErr := r.connection.Stream("sudo "+varsList+path, anonymousWriter{write: func(p []byte) (int, error) {
		n, buffErr := stdoutBuffer.Write(p)

		if label != "" {
			r.logger.Debug("bbr", "[%s] stdout: %s", label, string(p))
		} else {
			r.logger.Debug("bbr", "stdout: %s", string(p))
		}

		if buffErr != nil {
			return n, buffErr
		}

		return len(p), nil
	}})

	if label != "" {
		r.logger.Debug("bbr", "[%s] stderr: %s", label, string(stderr))
	} else {
		r.logger.Debug("bbr", "stderr: %s", string(stderr))
	}

	stdoutBytes := stdoutBuffer.Bytes()

	if runErr != nil {
		return "", runErr
	}

	if exitCode != 0 {
		return "", exitError(stderr, exitCode)
	}

	return string(stdoutBytes), nil
}

// anonymousWriter implements the Writer interface using a private
// 'write' function that you must set at construction time.
//
// This is useful for if you want a writer to behave like an anonymous
// function that can close over local state.
type anonymousWriter struct {
	write func(p []byte) (n int, err error)
}
func (w anonymousWriter) Write(p []byte) (n int, err error) {
	return w.write(p)
}

func (r SshRemoteRunner) FindFiles(pattern string) ([]string, error) {
	stdout, stderr, exitCode, err := r.connection.Run(fmt.Sprintf("sudo sh -c 'find %s -type f'", pattern))

	r.logOutput(stdout, stderr, "find files")

	if err != nil {
		return nil, err
	}

	if exitCode != 0 {
		if strings.Contains(string(stderr), "No such file or directory") {
			r.logger.Debug("bbr", "No files found for pattern '%s'", pattern)
			return []string{}, nil
		} else {
			return nil, exitError(stderr, exitCode)
		}
	}

	output := strings.TrimSpace(string(stdout))
	return strings.Split(output, "\n"), nil
}

func (r SshRemoteRunner) IsWindows() (bool, error) {
	stdout, _, _, err := r.connection.Run(`echo %OS%`)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(stdout)) == "Windows_NT", nil
}

func (r SshRemoteRunner) runOnInstance(cmd string) (string, error) {
	stdout, stderr, exitCode, runErr := r.connection.Run(cmd)

	err := r.logAndCheckErrors(stdout, stderr, exitCode, runErr, "")
	if err != nil {
		return "", err
	}

	return string(stdout), nil
}

func (r SshRemoteRunner) logAndCheckErrors(stdout, stderr []byte, exitCode int, err error, label string) error {
	r.logOutput(stdout, stderr, label)

	if err != nil {
		return err
	}

	if exitCode != 0 {
		return exitError(stderr, exitCode)
	}

	return nil
}

func (r SshRemoteRunner) logOutput(stdout []byte, stderr []byte, label string) {
	if label != "" {
		r.logger.Debug("bbr", "[%s] stdout: %s", label, string(stdout))
		r.logger.Debug("bbr", "[%s] stderr: %s", label, string(stderr))

	} else {
		r.logger.Debug("bbr", "stdout: %s", string(stdout))
		r.logger.Debug("bbr", "stderr: %s", string(stderr))
	}
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
