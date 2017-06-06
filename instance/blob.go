package instance

import (
	"fmt"
	"io"
	"strings"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o fakes/fake_ssh_connection.go . SSHConnection
type SSHConnection interface {
	Stream(cmd string, writer io.Writer) ([]byte, int, error)
	StreamStdin(cmd string, reader io.Reader) ([]byte, []byte, int, error)
	Run(cmd string) ([]byte, []byte, int, error)
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

func NewNamedBackupBlob(instance orchestrator.InstanceIdentifer, job Job, sshConn SSHConnection, logger Logger) *Blob {
	return &Blob{
		isNamed:           true,
		artifactDirectory: job.BackupArtifactDirectory(),
		name:              job.BackupBlobName(),
		instance:          instance,
		SSHConnection:     sshConn,
		Logger:            logger,
	}
}
func NewNamedRestoreBlob(instance orchestrator.InstanceIdentifer, job Job, sshConn SSHConnection, logger Logger) *Blob {
	return &Blob{
		isNamed:           true,
		artifactDirectory: job.RestoreArtifactDirectory(),
		name:              job.RestoreBlobName(),
		instance:          instance,
		SSHConnection:     sshConn,
		Logger:            logger,
	}
}

func NewDefaultBlob(name string, instance orchestrator.InstanceIdentifer, sshConn SSHConnection, logger Logger) *Blob {
	return &Blob{
		isNamed:           false,
		index:             instance.Index(),
		artifactDirectory: orchestrator.ArtifactDirectory,
		name:              name,
		instance:          instance,
		SSHConnection:     sshConn,
		Logger:            logger,
	}
}

type Blob struct {
	isNamed           bool
	index             string
	artifactDirectory string
	name              string
	instance          orchestrator.InstanceIdentifer
	SSHConnection
	Logger
}

func (b *Blob) StreamFromRemote(writer io.Writer) error {
	b.Logger.Debug("bbr", "Streaming backup from instance %s/%s", b.Name(), b.instance.ID())
	stderr, exitCode, err := b.Stream(fmt.Sprintf("sudo tar -C %s -c .", b.artifactDirectory), writer)

	b.Logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		b.Logger.Debug("bbr", "Error streaming backup from remote instance. Exit code %d, error %s", exitCode, err.Error())
		return errors.Wrap(err, fmt.Sprintf("Error streaming backup from remote instance. Exit code %d, error %s, stderr %s", exitCode, err, stderr))
	}

	if exitCode != 0 {
		return errors.Errorf("Streaming backup from remote instance returned %d. Error: %s", exitCode, stderr)
	}

	return nil
}

func (b *Blob) StreamToRemote(reader io.Reader) error {
	stdout, stderr, exitCode, err := b.logAndRun("sudo mkdir -p "+b.artifactDirectory, "create backup directory on remote")

	if err != nil {
		return err
	}

	if exitCode != 0 {
		return errors.Errorf("Creating backup directory on the remote returned %d. Error: %s", exitCode, stderr)
	}

	b.Logger.Debug("bbr", "Streaming backup to instance %s/%s", b.instance.Name(), b.instance.ID())
	stdout, stderr, exitCode, err = b.StreamStdin(fmt.Sprintf("sudo sh -c 'tar -C %s -x'", b.artifactDirectory), reader)

	b.Logger.Debug("bbr", "Stdout: %s", string(stdout))
	b.Logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		b.Logger.Debug("bbr", "Error streaming backup to remote instance. Exit code %d, error %s", exitCode, err.Error())
		return errors.Wrap(err, fmt.Sprintf("Error running instance backup scripts. Exit code %d, error %s, stderr %s", exitCode, err, stderr))
	}

	if exitCode != 0 {
		return errors.Errorf("Streaming backup to remote instance returned %d. Error: %s", exitCode, stderr)
	}

	return nil
}

func (b *Blob) Size() (string, error) {
	stdout, stderr, exitCode, err := b.logAndRun(fmt.Sprintf("sudo du -sh %s | cut -f1", b.artifactDirectory), "check backup size")
	if err != nil {
		return "", err
	}

	if exitCode != 0 {
		return "", errors.Errorf("Unable to check size of backup: %s", stderr)
	}

	return strings.TrimSpace(string(stdout)), nil
}

func (b *Blob) Checksum() (orchestrator.BackupChecksum, error) {
	b.Logger.Debug("bbr", "Calculating shasum for remote files on %s/%s", b.instance.Name(), b.instance.ID())

	stdout, stderr, exitCode, err := b.logAndRun(fmt.Sprintf("cd %s; sudo sh -c 'find . -type f | xargs shasum -a 256'", b.artifactDirectory), "checksum")
	if err != nil {
		return nil, err
	}

	if exitCode != 0 {
		return nil, errors.Errorf("instance checksum returned %d. Error: %s", exitCode, stderr)
	}

	return convertShasToMap(string(stdout)), nil
}

func (b *Blob) HasCustomName() bool {
	return b.isNamed
}

func (b *Blob) Name() string {
	return b.name
}

func (b *Blob) InstanceIndex() string {
	return b.instance.Index()
}

func (b *Blob) InstanceName() string {
	return b.instance.Name()
}

func (b *Blob) logAndRun(cmd, label string) ([]byte, []byte, int, error) {
	b.Logger.Debug("bbr", "Running %s on %s/%s", label, b.instance.Name(), b.instance.ID())

	stdout, stderr, exitCode, err := b.Run(cmd)
	b.Logger.Debug("bbr", "Stdout: %s", string(stdout))
	b.Logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		b.Logger.Debug("bbr", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, b.instance.Name(), b.instance.ID(), exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}

func (b *Blob) Delete() error {
	_, _, exitCode, err := b.logAndRun(fmt.Sprintf("sudo rm -rf %s", b.artifactDirectory), "deleting named blobs")

	if exitCode != 0 {
		return errors.Errorf("Error deleting blobs on instance %s/%s. Directory name %s. Exit code %d", b.instance.Name(), b.instance.ID(), b.artifactDirectory, exitCode)
	}

	return err
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
