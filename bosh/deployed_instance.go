package bosh

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/hashicorp/go-multierror"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
)

type DeployedInstance struct {
	director.Deployment
	InstanceGroupName             string
	BackupAndRestoreInstanceIndex string
	BoshInstanceID                string
	SSHConnection
	Logger
	instance.Jobs
}

func NewBoshInstance(instanceGroupName,
	instanceIndex,
	instanceID string,
	connection SSHConnection,
	deployment director.Deployment,
	logger Logger,
	jobs instance.Jobs,
) orchestrator.Instance {
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

func (d *DeployedInstance) IsPostBackupUnlockable() bool {
	return d.Jobs.AnyArePostBackupable()
}

func (d *DeployedInstance) IsPreBackupLockable() bool {
	return d.Jobs.AnyArePreBackupable()
}

func (d *DeployedInstance) CustomBlobNames() []string {
	return d.Jobs.BackupBlobNames()
}

func (d *DeployedInstance) PreBackupLock() error {

	var foundErrors error

	for _, job := range d.Jobs.PreBackupable() {
		d.Logger.Info("", "Locking %s on %s/%s for backup...", job.Name(), d.InstanceGroupName, d.BoshInstanceID)

		if err := d.runAndHandleErrs("pre backup lock", job.Name(), job.PreBackupScript()); err != nil {
			foundErrors = multierror.Append(foundErrors, err)
		}
		d.Logger.Info("", "Done.")
	}

	if foundErrors != nil {
		return foundErrors
	}

	return nil
}

func (d *DeployedInstance) Backup() error {

	var foundErrors error

	for _, job := range d.Jobs.Backupable() {
		d.Logger.Debug("", "> %s", job.BackupScript())
		d.Logger.Info("", "Backing up %s on %s/%s...", job.Name(), d.InstanceGroupName, d.BoshInstanceID)

		stdout, stderr, exitCode, err := d.logAndRun(
			fmt.Sprintf(
				"sudo mkdir -p %s && sudo ARTIFACT_DIRECTORY=%s/ %s",
				job.BackupArtifactDirectory(),
				job.BackupArtifactDirectory(),
				job.BackupScript(),
			),
			"backup",
		)

		if err := d.handleErrs(job.Name(), "backup", err, exitCode, stdout, stderr); err != nil {
			foundErrors = multierror.Append(foundErrors, err)
		}

		d.Logger.Info("", "Done.")
	}

	if foundErrors != nil {
		return foundErrors
	}

	return nil
}

func (d *DeployedInstance) PostBackupUnlock() error {

	var foundErrors error

	for _, job := range d.Jobs.PostBackupable() {
		d.Logger.Info("", "Unlocking %s on %s/%s...", job.Name(), d.InstanceGroupName, d.BoshInstanceID)

		if err := d.runAndHandleErrs("unlock", job.Name(), job.PostBackupScript()); err != nil {
			foundErrors = multierror.Append(foundErrors, err)
		}
		d.Logger.Info("", "Done.")
	}

	if foundErrors != nil {
		return foundErrors
	}

	return nil
}

func (d *DeployedInstance) Restore() error {
	var restoreErrors error

	for _, job := range d.Jobs.Restorable() {
		d.Logger.Debug("", "> %s", job.RestoreScript())
		d.Logger.Info("", "Restoring %s on %s/%s...", job.Name(), d.InstanceGroupName, d.BoshInstanceID)

		stdout, stderr, exitCode, err := d.logAndRun(
			fmt.Sprintf(
				"sudo ARTIFACT_DIRECTORY=%s/ %s",
				job.RestoreArtifactDirectory(),
				job.RestoreScript(),
			),
			"restore",
		)

		if err := d.handleErrs(job.Name(), "restore", err, exitCode, stdout, stderr); err != nil {
			restoreErrors = multierror.Append(restoreErrors, err)
		}
		d.Logger.Info("", "Done.")
	}

	if restoreErrors != nil {
		return restoreErrors
	}

	return nil
}

func (d *DeployedInstance) IsRestorable() bool {
	return d.Jobs.AnyAreRestorable()
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

func (d *DeployedInstance) BlobsToBackup() []orchestrator.BackupBlob {
	blobs := []orchestrator.BackupBlob{}

	for _, job := range d.Jobs.WithNamedBackupBlobs() {
		blobs = append(blobs, instance.NewNamedBackupBlob(d, job, d.SSHConnection, d.Logger))
	}

	if d.Jobs.AnyNeedDefaultBlobsForBackup() {
		blobs = append(blobs, instance.NewDefaultBlob(d, d.SSHConnection, d.Logger))
	}

	return blobs
}

func (d *DeployedInstance) BlobsToRestore() []orchestrator.BackupBlob {
	blobs := []orchestrator.BackupBlob{}

	if d.Jobs.AnyNeedDefaultBlobsForRestore() {
		blobs = append(blobs, instance.NewDefaultBlob(d, d.SSHConnection, d.Logger))
	}

	for _, job := range d.Jobs.WithNamedRestoreBlobs() {
		blobs = append(blobs, instance.NewNamedRestoreBlob(d, job, d.SSHConnection, d.Logger))
	}

	return blobs
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

func (d *DeployedInstance) Name() string {
	return d.InstanceGroupName
}

func (d *DeployedInstance) Index() string {
	return d.BackupAndRestoreInstanceIndex
}

func (d *DeployedInstance) ID() string {
	return d.BoshInstanceID
}

func (d *DeployedInstance) runAndHandleErrs(label, jobName string, script instance.Script) error {
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
