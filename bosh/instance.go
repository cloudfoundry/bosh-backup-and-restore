package bosh

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"bytes"
	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/hashicorp/go-multierror"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
)

type DeployedInstance struct {
	director.Deployment
	InstanceGroupName             string
	BackupAndRestoreInstanceIndex string
	BoshInstanceID                string
	SSHConnection
	Logger
	backupable *bool
	restorable *bool
	unlockable *bool
	lockable   *bool
	Jobs
}

//go:generate counterfeiter -o fakes/fake_ssh_connection.go . SSHConnection
type SSHConnection interface {
	Stream(cmd string, writer io.Writer) ([]byte, int, error)
	StreamStdin(cmd string, reader io.Reader) ([]byte, []byte, int, error)
	Run(cmd string) ([]byte, []byte, int, error)
	Cleanup() error
	Username() string
}

func NewBoshInstance(instanceGroupName,
	instanceIndex,
	instanceID string,
	connection SSHConnection,
	deployment director.Deployment,
	logger Logger,
	jobs Jobs,
) backuper.Instance {
	return &DeployedInstance{
		BackupAndRestoreInstanceIndex: instanceIndex,
		InstanceGroupName:             instanceGroupName,
		BoshInstanceID:                instanceID,
		SSHConnection:                 connection,
		Deployment:                    deployment,
		Logger:                        logger,
		Jobs:                          jobs,
	}
}

func (d *DeployedInstance) IsBackupable() bool {
	return d.Jobs.AnyAreBackupable()
}

func (d *DeployedInstance) IsPostBackupUnlockable() (bool, error) {
	if d.unlockable != nil {
		return *d.unlockable, nil
	}
	_, _, exitCode, err := d.logAndRun("sudo ls /var/vcap/jobs/*/bin/p-post-backup-unlock", "check for post-backup-unlock scripts")
	if err != nil {
		return false, err
	}
	unlockable := exitCode == 0
	d.unlockable = &unlockable

	return *d.unlockable, err
}

func (d *DeployedInstance) IsPreBackupLockable() (bool, error) {
	if d.lockable != nil {
		return *d.lockable, nil
	}
	_, _, exitCode, err := d.logAndRun("sudo ls /var/vcap/jobs/*/bin/p-pre-backup-lock", "check for pre-backup-lock scripts")
	if err != nil {
		return false, err
	}

	lockable := exitCode == 0
	d.lockable = &lockable

	return *d.lockable, err
}

func (d *DeployedInstance) PreBackupLock() error {
	d.Logger.Info("", "Locking %s/%s for backup...", d.InstanceGroupName, d.BoshInstanceID)

	var foundErrors error

	for _, job := range d.Jobs.PreBackupable() {
		if err := d.runAndHandleErrs("pre backup lock", job.Name(), job.PreBackupScript()); err != nil {
			foundErrors = multierror.Append(foundErrors, err)
		}
	}

	if foundErrors != nil {
		return foundErrors
	}

	d.Logger.Info("", "Done.")
	return nil
}

func (d *DeployedInstance) Backup() error {
	d.Logger.Info("", "Backing up %s/%s...", d.InstanceGroupName, d.BoshInstanceID)

	var foundErrors error

	for _, job := range d.Jobs.Backupable() {
		d.Logger.Debug("", "> %s", job.BackupScript())

		stdout, stderr, exitCode, err := d.logAndRun(
			fmt.Sprintf(
				"sudo mkdir -p %s && sudo ARTIFACT_DIRECTORY=%s/ %s",
				job.ArtifactDirectory(),
				job.ArtifactDirectory(),
				job.BackupScript(),
			),
			"backup",
		)

		if err := d.handleErrs(job.Name(), "backup", err, exitCode, stdout, stderr); err != nil {
			foundErrors = multierror.Append(foundErrors, err)
		}
	}

	if foundErrors != nil {
		return foundErrors
	}

	d.Logger.Info("", "Done.")
	return nil
}

func (d *DeployedInstance) PostBackupUnlock() error {
	d.Logger.Info("", "Unlocking %s/%s...", d.InstanceGroupName, d.BoshInstanceID)

	var foundErrors error

	for _, job := range d.Jobs.PostBackupable() {
		if err := d.runAndHandleErrs("unlock", job.Name(), job.PostBackupScript()); err != nil {
			foundErrors = multierror.Append(foundErrors, err)
		}
	}

	if foundErrors != nil {
		return foundErrors
	}

	d.Logger.Info("", "Done.")
	return nil
}

func (d *DeployedInstance) Restore() error {
	d.Logger.Info("", "Restoring to %s/%s...", d.InstanceGroupName, d.BoshInstanceID)

	var restoreErrors error

	for _, job := range d.Jobs.Restorable() {
		d.Logger.Debug("", "> %s", job.RestoreScript())

		artifactDirectory := fmt.Sprintf("/var/vcap/store/backup/%s", job.Name())
		stdout, stderr, exitCode, err := d.logAndRun(
			fmt.Sprintf(
				"sudo ARTIFACT_DIRECTORY=%s/ %s",
				artifactDirectory,
				job.RestoreScript(),
			),
			"restore",
		)

		if err := d.handleErrs(job.Name(), "restore", err, exitCode, stdout, stderr); err != nil {
			restoreErrors = multierror.Append(restoreErrors, err)
		}
	}

	if restoreErrors != nil {
		return restoreErrors
	}

	d.Logger.Info("", "Done.")
	return nil
}

func (d *DeployedInstance) StreamBackupToRemote(reader io.Reader) error {
	stdout, stderr, exitCode, err := d.logAndRun("sudo mkdir -p /var/vcap/store/backup/", "create backup directory on remote")

	if err != nil {
		return err
	}

	if exitCode != 0 {
		return fmt.Errorf("Creating backup directory on the remote returned %d. Error: %s", exitCode, stderr)
	}

	d.Logger.Debug("", "Streaming backup to instance %s/%s", d.InstanceGroupName, d.BoshInstanceID)
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

func (d *DeployedInstance) BackupChecksum() (backuper.BackupChecksum, error) {
	stdout, stderr, exitCode, err := d.logAndRun("cd /var/vcap/store/backup; sudo sh -c 'find . -type f | xargs shasum'", "checksum")

	if err != nil {
		return nil, err
	}

	if exitCode != 0 {
		return nil, fmt.Errorf("Instance checksum returned %d. Error: %s", exitCode, stderr)
	}

	return convertShasToMap(string(stdout)), nil
}

func (d *DeployedInstance) IsRestorable() (bool, error) {
	if d.restorable != nil {
		return *d.restorable, nil
	}
	_, _, exitCode, err := d.logAndRun("ls /var/vcap/jobs/*/bin/p-restore", "check for restore scripts")
	if err != nil {
		return false, err
	}

	restorable := exitCode == 0
	d.restorable = &restorable

	return *d.restorable, err
}

func (d *DeployedInstance) BackupSize() (string, error) {
	stdout, stderr, exitCode, err := d.logAndRun("sudo du -sh /var/vcap/store/backup/ | cut -f1", "check backup size")

	if exitCode != 0 {
		return "", fmt.Errorf("Unable to check size of backup: %s", stderr)
	}

	size := strings.TrimSpace(string(stdout))
	return size, err
}

func (d *DeployedInstance) Cleanup() error {
	var errs error
	d.Logger.Debug("", "Cleaning up SSH connection on instance %s %s", d.InstanceGroupName, d.BoshInstanceID)
	removeArtifactError := d.removeBackupArtifacts()
	if removeArtifactError != nil {
		errs = multierror.Append(errs, removeArtifactError)
	}
	cleanupSSHError := d.CleanUpSSH(director.NewAllOrInstanceGroupOrInstanceSlug(d.InstanceGroupName, d.BoshInstanceID), director.SSHOpts{Username: d.SSHConnection.Username()})
	if cleanupSSHError != nil {
		errs = multierror.Append(errs, cleanupSSHError)
	}
	return errs
}

func (d *DeployedInstance) RemoteArtifact() backuper.RemoteArtifact {
	return &DefaultRemoteArtifact{
		Instance: d,
		SSHConnection: d.SSHConnection,
		Logger: d.Logger,
	}
}

func (d *DeployedInstance) logAndRun(cmd, label string) ([]byte, []byte, int, error) {
	d.Logger.Debug("", "Running %s on %s/%s", label, d.InstanceGroupName, d.BoshInstanceID)

	stdout, stderr, exitCode, err := d.Run(cmd)
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, d.InstanceGroupName, d.BoshInstanceID, exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}

func (d *DeployedInstance) filesPresent(path string) {
	d.Logger.Debug("", "Listing contents of %s on %s/%s", path, d.InstanceGroupName, d.BoshInstanceID)

	stdout, _, _, _ := d.Run("sudo ls " + path)
	stdout = bytes.TrimSpace(stdout)
	files := strings.Split(string(stdout), "\n")

	for _, f := range files {
		d.Logger.Debug("", "> %s", f)
	}
}

func (d *DeployedInstance) Name() string {
	return d.InstanceGroupName
}

func (d *DeployedInstance) Index() string {
	return d.BackupAndRestoreInstanceIndex
}

func (d *DeployedInstance) ID() string {
	return d.BoshInstanceID
}

func (d *DeployedInstance) runAndHandleErrs(label, jobName string, script Script) error {
	d.Logger.Debug("", "> %s", script)

	stdout, stderr, exitCode, err := d.logAndRun(
		fmt.Sprintf(
			"sudo %s",
			script,
		),
		label,
	)

	return d.handleErrs(jobName, label, err, exitCode, stdout, stderr)
}

func (d *DeployedInstance) handleErrs(jobName, label string, err error, exitCode int, stdout, stderr []byte) error {
	var foundErrors error

	if err != nil {
		d.Logger.Error("", fmt.Sprintf(
			"Error attempting to run %s script for job %s on %s/%s. Error: %s",
			label,
			jobName,
			d.InstanceGroupName,
			d.BoshInstanceID,
			err.Error(),
		))
		foundErrors = multierror.Append(foundErrors, err)
	}

	if exitCode != 0 {
		errorString := fmt.Sprintf(
			"%s script for job %s failed on %s/%s.\nStdout: %s\nStderr: %s",
			label,
			jobName,
			d.InstanceGroupName,
			d.BoshInstanceID,
			stdout,
			stderr,
		)

		foundErrors = multierror.Append(foundErrors, errors.New(errorString))

		d.Logger.Error("", errorString)
	}

	return foundErrors
}

func (d *DeployedInstance) removeBackupArtifacts() error {
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

