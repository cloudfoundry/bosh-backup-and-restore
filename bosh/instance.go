package bosh

import (
	"fmt"
	"io"
	"strings"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/hashicorp/go-multierror"
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
	StreamStdin(cmd string, reader io.Reader) ([]byte, []byte, int, error)
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
	stdin, stdout, exitCode, err := d.Run("ls /var/vcap/jobs/*/bin/p-backup")

	d.Logger.Debug("", "Stdin: %s", string(stdin))
	d.Logger.Debug("", "Stdout: %s", string(stdout))

	if err != nil {
		d.Logger.Debug("", "Error checking instance has backup scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	return exitCode == 0, err
}

func (d DeployedInstance) Backup() error {
	d.Logger.Info("", "Backing up %s-%s...", d.InstanceGroupName, d.InstanceIndex)
	d.Logger.Debug("", "Running all backup scripts on instance %s %s", d.InstanceGroupName, d.InstanceIndex)
	stdout, stderr, exitCode, err := d.Run("sudo mkdir -p /var/vcap/store/backup && ls /var/vcap/jobs/*/bin/p-backup | xargs -IN sudo sh -c N")

	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running instance backup scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	if exitCode != 0 {
		return fmt.Errorf("Instance backup scripts returned %d. Error: %s", exitCode, stderr)
	}

	d.Logger.Info("", "Done.")
	return err
}

func (d DeployedInstance) Restore() error {
	stdout, stderr, exitCode, err := d.Run("ls /var/vcap/jobs/*/bin/p-restore | xargs -IN sudo sh -c N")

	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running instance restore scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	if exitCode != 0 {
		return fmt.Errorf("Instance restore scripts returned %d. Error: %s", exitCode, stderr)
	}

	return err
}

func (d DeployedInstance) StreamBackupFromRemote(writer io.Writer) error {
	d.Logger.Debug("", "Streaming backup from instance %s %s", d.InstanceGroupName, d.InstanceIndex)
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

func (d DeployedInstance) StreamBackupToRemote(reader io.Reader) error {
	d.Logger.Debug("", "Creating backup directory on instance %s %s", d.InstanceGroupName, d.InstanceIndex)
	stdout, stderr, exitCode, err := d.Run("sudo mkdir -p /var/vcap/store/backup/")

	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error creating backup directory on remote instance. Exit code %d, error %s", exitCode, err.Error())
		return err
	}

	if exitCode != 0 {
		return fmt.Errorf("Creating backup directory on the remote returned %d. Error: %s", exitCode, stderr)
	}

	d.Logger.Debug("", "Streaming backup to instance %s %s", d.InstanceGroupName, d.InstanceIndex)
	stdout, stderr, exitCode, err = d.StreamStdin("sudo sh -c 'tar -C /var/vcap/store/backup -zx'", reader)

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

func (d DeployedInstance) BackupChecksum() (backuper.BackupChecksum, error) {
	d.Logger.Debug("", "Running checksum on instance %s %s", d.InstanceGroupName, d.InstanceIndex)

	stdout, stderr, exitCode, err := d.Run("cd /var/vcap/store/backup; sudo sh -c 'find . -type f | xargs shasum'")

	if err != nil {
		d.Logger.Debug("", "Error generating checksum. Exit code %d, error %s", exitCode, err.Error())
		return nil, err
	}

	if exitCode != 0 {
		return nil, fmt.Errorf("Instance checksum returned %d. Error: %s", exitCode, stderr)
	}

	return convertShasToMap(string(stdout)), nil
}

func (d DeployedInstance) IsRestorable() (bool, error) {
	d.Logger.Debug("", "Checking instance %s %s has restore scripts", d.InstanceGroupName, d.InstanceIndex)
	stdout, stderr, exitCode, err := d.Run("ls /var/vcap/jobs/*/bin/p-restore")

	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error checking instance has backup scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	return exitCode == 0, err
}

func (d DeployedInstance) BackupSize() (string, error) {
	stdout, stderr, exitCode, err := d.Run("sudo du -sh /var/vcap/store/backup/ | cut -f1")

	if err != nil {
		d.Logger.Debug("", "Error checking size of backup. Exit code %d, error %s", exitCode, err.Error())
	}

	if exitCode != 0 {
		return "", fmt.Errorf("Unable to check size of backup: %s", stderr)
	}

	size := strings.TrimSpace(string(stdout))
	return size, nil
}

func (d DeployedInstance) Cleanup() error {
	var errs error
	d.Logger.Debug("", "Cleaning up SSH connection on instance %s %s", d.InstanceGroupName, d.InstanceIndex)
	removeArtifactError := d.removeBackupArtifacts()
	if removeArtifactError != nil {
		errs = multierror.Append(errs, removeArtifactError)
	}

	cleanupSSHError := d.CleanUpSSH(director.NewAllOrInstanceGroupOrInstanceSlug(d.InstanceGroupName, d.InstanceIndex), director.SSHOpts{Username: d.SSHConnection.Username()})
	if cleanupSSHError != nil {
		errs = multierror.Append(errs, cleanupSSHError)
	}
	return errs
}

func (d DeployedInstance) Name() string {
	return d.InstanceGroupName
}

func (d DeployedInstance) ID() string {
	return d.InstanceIndex
}

func (d DeployedInstance) removeBackupArtifacts() error {
	_, _, _, err := d.Run("sudo rm -rf /var/vcap/store/backup")
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
