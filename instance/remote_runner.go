package instance

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"strings"
)

//go:generate counterfeiter -o fakes/fake_remote_runner.go . RemoteRunner
type RemoteRunner interface {
	ConnectedUsername() string
	DirectoryExists(dir string) (bool, error)
	RemoveDirectory(dir string) error
	CompressDirectory(directory string, writer io.Writer) error
	CreateDirectory(directory string) error
	ExtractArchive(reader io.Reader, directory string) error
	SizeOf(path string) (string, error)
	ChecksumDirectory(path string) (map[string]string, error)
}

type SshRemoteRunner struct {
	instanceIdentifier InstanceIdentifier
	logger             Logger
	connection         ssh.SSHConnection
}

func NewRemoteRunner(sshConnection ssh.SSHConnection, instanceId InstanceIdentifier, logger Logger) SshRemoteRunner {
	return SshRemoteRunner{
		connection:         sshConnection,
		instanceIdentifier: instanceId,
		logger:             logger,
	}
}

func (r SshRemoteRunner) ConnectedUsername() string {
	return r.connection.Username()
}

func (r SshRemoteRunner) DirectoryExists(dir string) (bool, error) {
	_, _, exitCode, err := r.connection.Run(fmt.Sprintf("stat %s", dir))
	return exitCode == 0, err
}

func (r SshRemoteRunner) RemoveDirectory(dir string) error {
	_, err := r.runOnInstance(fmt.Sprintf("sudo rm -rf %s", dir))
	return err
}

func (r SshRemoteRunner) CompressDirectory(directory string, writer io.Writer) error {
	stderr, exitCode, err := r.connection.Stream(fmt.Sprintf("sudo tar -C %s -c .", directory), writer)
	return r.logAndCheckErrors([]byte{}, stderr, exitCode, err)
}

func (r SshRemoteRunner) CreateDirectory(directory string) error {
	_, err := r.runOnInstance("sudo mkdir -p "+directory)
	return err
}

func (r SshRemoteRunner) ExtractArchive(reader io.Reader, directory string) error {
	r.logger.Debug("bbr", "Streaming backup to instance %s", r.instanceIdentifier)

	stdout, stderr, exitCode, err := r.connection.StreamStdin(fmt.Sprintf("sudo sh -c 'tar -C %s -x'", directory), reader)

	return r.logAndCheckErrors(stdout, stderr, exitCode, err)
}

func (r SshRemoteRunner) SizeOf(path string) (string, error) {
	stdout, err := r.runOnInstance(fmt.Sprintf("sudo du -sh %s | cut -f1", path))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(stdout)), nil
}

func (r SshRemoteRunner) ChecksumDirectory(path string) (map[string]string, error) {
	stdout, err := r.runOnInstance(fmt.Sprintf("sudo sh -c 'cd %s && find . -type f | xargs shasum -a 256'", path))
	if err != nil {
		return nil, err
	}

	return convertShasToMap(stdout), nil
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
		r.logger.Debug("bbr", "Error running %s on instance %s. Exit code %d, error: %s", r.instanceIdentifier, exitCode, err.Error())
		return err
	}

	if exitCode != 0 {
		return errors.New(fmt.Sprintf("%s - exit code %d", string(stderr), exitCode))
	}

	return nil
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
