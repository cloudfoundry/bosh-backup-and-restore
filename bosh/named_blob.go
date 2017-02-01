package bosh

import (
	"fmt"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"io"
)

func NewNamedBlob(instance backuper.Instance, job Job, sshConn SSHConnection, logger Logger) *NamedBlob {
	return &NamedBlob{
		Job:           job,
		Instance:      instance,
		SSHConnection: sshConn,
		Logger:        logger,
	}
}

type NamedBlob struct {
	Job Job
	backuper.Instance
	SSHConnection
	Logger
}

func (d *NamedBlob) StreamFromRemote(writer io.Writer) error {
	d.Logger.Debug("", "Streaming backup from instance %s/%s", d.Name(), d.ID())
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

func (d *NamedBlob) BackupChecksum() (backuper.BackupChecksum, error) {
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
	return d.Job.blobName
}

func (d *NamedBlob) logAndRun(cmd, label string) ([]byte, []byte, int, error) {
	d.Logger.Debug("", "Running %s on %s/%s", label, d.Name(), d.ID())

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
