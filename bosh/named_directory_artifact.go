package bosh

import (
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"io"
)

func NewNamedRemoteArtifact(instance backuper.Instance, job Job, sshConn SSHConnection, logger Logger) *NamedRemoteArtifact {
	return &NamedRemoteArtifact{
		Job:           job,
		Instance:      instance,
		SSHConnection: sshConn,
		Logger:        logger,
	}
}

type NamedRemoteArtifact struct {
	Job Job
	backuper.Instance
	SSHConnection
	Logger
}

func (d *NamedRemoteArtifact) StreamFromRemote(writer io.Writer) error {
	return nil
}

func (d *NamedRemoteArtifact) logAndRun(cmd, label string) ([]byte, []byte, int, error) {
	d.Logger.Debug("", "Running %s on %s/%s", label, d.Name(), d.ID())

	stdout, stderr, exitCode, err := d.Run(cmd)
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, d.Name(), d.ID(), exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}
