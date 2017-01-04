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
	_, _, exitCode, err := d.logAndRun("ls /var/vcap/jobs/*/bin/p-backup", "check for backup scripts")

	return exitCode == 0, err
}

func (d DeployedInstance) PreBackupLock() error {
	_, stderr, exitCode, err := d.logAndRun("sudo ls /var/vcap/jobs/*/bin/p-pre-backup-lock | xargs -IN sudo sh -c N", "pre-backup-lock")

	if exitCode != 0 {
		return fmt.Errorf("Instance pre-backup-lock scripts returned %d. Error: %s", exitCode, stderr)
	}

	return err
}

func (d DeployedInstance) Backup() error {
	d.Logger.Info("", "Backing up %s-%s...", d.InstanceGroupName, d.InstanceIndex)

	_,stderr, exitCode, err := d.logAndRun("sudo mkdir -p /var/vcap/store/backup && ls /var/vcap/jobs/*/bin/p-backup | xargs -IN sudo sh -c N", "backup")

	if exitCode != 0 {
		return fmt.Errorf("Instance backup scripts returned %d. Error: %s", exitCode, stderr)
	}

	d.Logger.Info("", "Done.")
	return err
}

func (d DeployedInstance) Restore() error {
	_, stderr, exitCode, err := d.logAndRun("ls /var/vcap/jobs/*/bin/p-restore | xargs -IN sudo sh -c N", "restore")

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
	stdout, stderr, exitCode, err := d.logAndRun("sudo mkdir -p /var/vcap/store/backup/", "create backup directory on remote")

	if err != nil {
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
	stdout, stderr, exitCode, err := d.logAndRun("cd /var/vcap/store/backup; sudo sh -c 'find . -type f | xargs shasum'", "checksum")

	if err != nil {
		return nil, err
	}

	if exitCode != 0 {
		return nil, fmt.Errorf("Instance checksum returned %d. Error: %s", exitCode, stderr)
	}

	return convertShasToMap(string(stdout)), nil
}

func (d DeployedInstance) IsRestorable() (bool, error) {
	_, _, exitCode, err := d.logAndRun("ls /var/vcap/jobs/*/bin/p-restore", "check for restore scripts")

	return exitCode == 0, err
}

func (d DeployedInstance) BackupSize() (string, error) {
	stdout, stderr, exitCode, _ := d.logAndRun("sudo du -sh /var/vcap/store/backup/ | cut -f1", "check backup size")

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


func (d DeployedInstance) logAndRun(cmd, label string) ([]byte,[]byte, int, error) {
	d.Logger.Debug("", "Running %s on %s %s",label, d.InstanceGroupName, d.InstanceIndex)

	stdout, stderr, exitCode, err := d.Run(cmd)
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running %s. Exit code %d, error %s", label, exitCode, err.Error())
	}

	return stdout,stderr, exitCode, err
}

func (d DeployedInstance) Name() string {
	return d.InstanceGroupName
}

func (d DeployedInstance) ID() string {
	return d.InstanceIndex
}

func (d DeployedInstance) removeBackupArtifacts() error {
	_, _, _, err := d.logAndRun("sudo rm -rf /var/vcap/store/backup", "remove backup artifacts")
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
