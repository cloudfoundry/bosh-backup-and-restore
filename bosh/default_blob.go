package bosh

import (
	"fmt"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
	"io"
)

func NewDefaultBlob(instance backuper.Instance, sshConn SSHConnection, logger Logger) *DefaultBlob {
	return &DefaultBlob{
		Instance:      instance,
		SSHConnection: sshConn,
		Logger:        logger,
	}
}

type DefaultBlob struct {
	backuper.Instance
	SSHConnection
	Logger
}

func (d *DefaultBlob) StreamFromRemote(writer io.Writer) error {
	d.Logger.Debug("", "Streaming backup from instance %s/%s", d.Name(), d.ID())
	stderr, exitCode, err := d.Stream("sudo tar -C /var/vcap/store/backup -zc .", writer)

	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running instance backup scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	if exitCode != 0 {
		return fmt.Errorf("Instance backup scripts returned %d. Error: %s", exitCode, stderr)
	}

	return err
}

func (d *DefaultBlob) logAndRun(cmd, label string) ([]byte, []byte, int, error) {
	d.Logger.Debug("", "Running %s on %s/%s", label, d.Name(), d.ID())

	stdout, stderr, exitCode, err := d.Run(cmd)
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, d.Name(), d.ID(), exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}
