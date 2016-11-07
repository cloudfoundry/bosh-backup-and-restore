package bosh

import (
	"fmt"
	"io"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
)

type DeployedInstance struct {
	director.Deployment
	InstanceGroupName string
	InstanceIndex     string
	SSHConnection
	Logger
}

//go:generate counterfeiter -o fakes/fake_ssh_connection.go . SSHConnection
type SSHConnection interface {
	Stream(cmd string, writer io.Writer) ([]byte, int, error)
	Run(cmd string) ([]byte, []byte, int, error)
	Cleanup() error
	Username() string
}

func NewBoshInstance(instanceGroupName, instanceIndex string, connection SSHConnection, deployment director.Deployment, logger Logger) backuper.Instance {
	return DeployedInstance{
		InstanceIndex:     instanceIndex,
		InstanceGroupName: instanceGroupName,
		SSHConnection:     connection,
		Deployment:        deployment,
		Logger:            logger,
	}
}

func (d DeployedInstance) IsBackupable() (bool, error) {
	d.Logger.Debug("", "Checking instance %s %s has backup scripts", d.InstanceGroupName, d.InstanceIndex)
	stdin, stdout, exitCode, err := d.Run("ls /var/vcap/jobs/*/bin/backup")

	d.Logger.Debug("", "Stdin: %s", string(stdin))
	d.Logger.Debug("", "Stdout: %s", string(stdout))

	if err != nil {
		d.Logger.Debug("", "Error checking instance has backup scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	return exitCode == 0, err
}

func (d DeployedInstance) Backup() error {
	d.Logger.Debug("", "Running all backup scripts on instance %s %s", d.InstanceGroupName, d.InstanceIndex)
	stdout, stderr, exitCode, err := d.Run("sudo mkdir -p /var/vcap/store/backup && ls /var/vcap/jobs/*/bin/backup | xargs -IN sudo sh -c N")

	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running instance backup scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	if exitCode != 0 {
		return fmt.Errorf("Instance backup scripts returned %d. Error: %s", exitCode, stderr)
	}

	return err
}

func (d DeployedInstance) StreamBackupTo(writer io.Writer) error {
	d.Logger.Debug("", "Running all backup scripts on instance %s %s", d.InstanceGroupName, d.InstanceIndex)
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

func (d DeployedInstance) IsRestorable() (bool, error) {
	d.Logger.Debug("", "Checking instance %s %s has restore scripts", d.InstanceGroupName, d.InstanceIndex)
	stdout, stderr, exitCode, err := d.Run("ls /var/vcap/jobs/*/bin/restore")

	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error checking instance has backup scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	return exitCode == 0, err
}

func (d DeployedInstance) Cleanup() error {
	d.Logger.Debug("", "Cleaning up SSH connection on instance %s %s", d.InstanceGroupName, d.InstanceIndex)
	return d.CleanUpSSH(director.NewAllOrPoolOrInstanceSlug(d.InstanceGroupName, d.InstanceIndex), director.SSHOpts{Username: d.SSHConnection.Username()})
}

func (d DeployedInstance) Name() string {
	return d.InstanceGroupName
}

func (d DeployedInstance) ID() string {
	return d.InstanceIndex
}
