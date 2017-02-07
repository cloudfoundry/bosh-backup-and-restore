package instance

import (
	"fmt"
	"io"
	"strings"

	"github.com/pivotal-cf/pcf-backup-and-restore/orchestrator"
)

//go:generate counterfeiter -o fakes/fake_ssh_connection.go . SSHConnection
type SSHConnection interface {
	Stream(cmd string, writer io.Writer) ([]byte, int, error)
	StreamStdin(cmd string, reader io.Reader) ([]byte, []byte, int, error)
	Run(cmd string) ([]byte, []byte, int, error)
}

type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

func NewNamedBlob(instance orchestrator.Instance, job Job, sshConn SSHConnection, logger Logger) *NamedBlob {
	return &NamedBlob{
		Job:           job,
		Instance:      instance,
		SSHConnection: sshConn,
		Logger:        logger,
	}
}

type NamedBlob struct {
	Job      Job
	Instance orchestrator.Instance
	SSHConnection
	Logger
}

func (d *NamedBlob) StreamFromRemote(writer io.Writer) error {
	d.Logger.Debug("", "Streaming backup from instance %s/%s", d.Name(), d.Instance.ID())
	stderr, exitCode, err := d.Stream(fmt.Sprintf("sudo tar -C %s -zc .", d.Job.ArtifactDirectory()), writer)

	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running instance backup scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	if exitCode != 0 {
		return fmt.Errorf("Instance backup scripts returned %d. Error: %s", exitCode, stderr)
	}

	return err
}

func (d *NamedBlob) StreamBackupToRemote(reader io.Reader) error {
	stdout, stderr, exitCode, err := d.logAndRun("sudo mkdir -p "+d.Job.ArtifactDirectory(), "create backup directory on remote")

	if err != nil {
		return err
	}

	if exitCode != 0 {
		return fmt.Errorf("Creating backup directory on the remote returned %d. Error: %s", exitCode, stderr)
	}

	d.Logger.Debug("", "Streaming backup to instance %s/%s", d.Instance.Name(), d.Instance.ID())
	stdout, stderr, exitCode, err = d.StreamStdin(fmt.Sprintf("sudo sh -c 'tar -C %s -zx'", d.Job.ArtifactDirectory()), reader)

	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error streaming backup to remote instance. Exit code %d, error %s", exitCode, err.Error())
	}

	if exitCode != 0 {
		return fmt.Errorf("Streaming backup to remote returned %d. Error: %s", exitCode, stderr)
	}

	return err
}

func (d *NamedBlob) BackupSize() (string, error) {
	stdout, stderr, exitCode, err := d.logAndRun(fmt.Sprintf("sudo du -sh %s | cut -f1", d.Job.ArtifactDirectory()), "check backup size")

	if exitCode != 0 {
		return "", fmt.Errorf("Unable to check size of backup: %s", stderr)
	}

	size := strings.TrimSpace(string(stdout))
	return size, err
}

func (d *NamedBlob) BackupChecksum() (orchestrator.BackupChecksum, error) {
	stdout, stderr, exitCode, err := d.logAndRun(fmt.Sprintf("cd %s; sudo sh -c 'find . -type f | xargs shasum'", d.Job.ArtifactDirectory()), "checksum")

	if err != nil {
		return nil, err
	}

	if exitCode != 0 {
		return nil, fmt.Errorf("Instance checksum returned %d. Error: %s", exitCode, stderr)
	}

	return convertShasToMap(string(stdout)), nil
}

func (d *NamedBlob) IsNamed() bool {
	return true
}

func (d *NamedBlob) Name() string {
	return d.Job.BlobName()
}

func (d *NamedBlob) ID() string {
	return d.Instance.ID()
}

func (d *NamedBlob) Index() string {
	return ""
}

func (d *NamedBlob) logAndRun(cmd, label string) ([]byte, []byte, int, error) {
	d.Logger.Debug("", "Running %s on %s/%s", label, d.Instance.Name(), d.Instance.ID())

	stdout, stderr, exitCode, err := d.Run(cmd)
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, d.Instance.Name(), d.Instance.ID(), exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}

func (d *NamedBlob) Delete() error {
	_, _, exitCode, err := d.logAndRun(fmt.Sprintf("sudo rm -rf %s", d.Job.ArtifactDirectory()), "deleting named blobs")

	if exitCode != 0 {
		return fmt.Errorf("Error deleting blobs on instance %s/%s. Directory name %s. Exit code %d", d.Instance.Name(), d.Instance.ID(), d.Job.ArtifactDirectory(), exitCode)
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
