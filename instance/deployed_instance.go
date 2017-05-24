package instance

import (
	"errors"
	"fmt"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
)

type DeployedInstance struct {
	backupAndRestoreInstanceIndex string
	instanceID                    string
	instanceGroupName             string
	ssh.SSHConnection
	Logger
	Jobs
}

func NewDeployedInstance(instanceIndex string, instanceGroupName string, instanceID string, connection ssh.SSHConnection, logger Logger, jobs Jobs) *DeployedInstance {
	deployedInstance := &DeployedInstance{
		backupAndRestoreInstanceIndex: instanceIndex,
		instanceGroupName:             instanceGroupName,
		instanceID:                    instanceID,
		SSHConnection:                 connection,
		Logger:                        logger,
		Jobs:                          jobs,
	}
	return deployedInstance
}

func (d *DeployedInstance) ArtifactDirExists() bool {
	_, _, exitCode, _ := d.RunOnInstance(
		fmt.Sprintf(
			"stat %s",
			orchestrator.ArtifactDirectory,
		),
		"artifact directory check",
	)

	return exitCode == 0
}

func (d *DeployedInstance) HasBackupScript() bool {
	return d.Jobs.AnyAreBackupable()
}

func (d *DeployedInstance) IsPostBackupUnlockable() bool {
	return d.Jobs.AnyArePostBackupable()
}

func (d *DeployedInstance) IsPreBackupLockable() bool {
	return d.Jobs.AnyArePreBackupable()
}

func (d *DeployedInstance) CustomBackupBlobNames() []string {
	return d.Jobs.CustomBackupBlobNames()
}

func (d *DeployedInstance) CustomRestoreBlobNames() []string {
	return d.Jobs.CustomRestoreBlobNames()
}

func (d *DeployedInstance) PreBackupLock() error {

	var foundErrors []error

	for _, job := range d.Jobs.PreBackupable() {
		d.Logger.Info("", "Locking %s on %s/%s for backup...", job.Name(), d.instanceGroupName, d.instanceID)

		if err := d.runAndHandleErrs("pre backup lock", job.Name(), job.PreBackupScript()); err != nil {
			foundErrors = append(foundErrors, err)
		}
		d.Logger.Info("", "Done.")
	}

	return orchestrator.ConvertErrors(foundErrors)
}

func (d *DeployedInstance) Backup() error {

	var foundErrors []error

	for _, job := range d.Jobs.Backupable() {
		d.Logger.Debug("", "> %s", job.BackupScript())
		d.Logger.Info("", "Backing up %s on %s/%s...", job.Name(), d.instanceGroupName, d.instanceID)

		stdout, stderr, exitCode, err := d.RunOnInstance(
			fmt.Sprintf(
				"sudo mkdir -p %s && sudo %s %s",
				job.BackupArtifactDirectory(),
				artifactDirectoryVariables(job.BackupArtifactDirectory()),
				job.BackupScript(),
			),
			"backup",
		)

		if err := d.handleErrs(job.Name(), "backup", err, exitCode, stdout, stderr); err != nil {
			foundErrors = append(foundErrors, err)
		}

		d.Logger.Info("", "Done.")
	}

	return orchestrator.ConvertErrors(foundErrors)
}
func artifactDirectoryVariables(artifactDirectory string) string {
	return fmt.Sprintf("BBR_ARTIFACT_DIRECTORY=%s/ ARTIFACT_DIRECTORY=%[1]s/", artifactDirectory)
}

func (d *DeployedInstance) PostBackupUnlock() error {

	var foundErrors []error

	for _, job := range d.Jobs.PostBackupable() {
		d.Logger.Info("", "Unlocking %s on %s/%s...", job.Name(), d.instanceGroupName, d.instanceID)

		if err := d.runAndHandleErrs("unlock", job.Name(), job.PostBackupScript()); err != nil {
			foundErrors = append(foundErrors, err)
		}
		d.Logger.Info("", "Done.")
	}

	return orchestrator.ConvertErrors(foundErrors)
}

func (d *DeployedInstance) Restore() error {
	var restoreErrors []error

	for _, job := range d.Jobs.Restorable() {
		d.Logger.Debug("", "> %s", job.RestoreScript())
		d.Logger.Info("", "Restoring %s on %s/%s...", job.Name(), d.instanceGroupName, d.instanceID)

		stdout, stderr, exitCode, err := d.RunOnInstance(
			fmt.Sprintf(
				"sudo %s %s",
				artifactDirectoryVariables(job.RestoreArtifactDirectory()),
				job.RestoreScript(),
			),
			"restore",
		)

		if err := d.handleErrs(job.Name(), "restore", err, exitCode, stdout, stderr); err != nil {
			restoreErrors = append(restoreErrors, err)
		}
		d.Logger.Info("", "Done.")
	}

	return orchestrator.ConvertErrors(restoreErrors)
}

func (d *DeployedInstance) IsRestorable() bool {
	return d.Jobs.AnyAreRestorable()
}

func (d *DeployedInstance) BlobsToBackup() []orchestrator.BackupBlob {
	blobs := []orchestrator.BackupBlob{}

	for _, job := range d.Jobs.WithNamedBackupBlobs() {
		blobs = append(blobs, NewNamedBackupBlob(d, job, d.SSHConnection, d.Logger))
	}

	if d.Jobs.AnyNeedDefaultBlobsForBackup() {
		blobs = append(blobs, NewDefaultBlob(d, d.SSHConnection, d.Logger))
	}

	return blobs
}

func (d *DeployedInstance) BlobsToRestore() []orchestrator.BackupBlob {
	blobs := []orchestrator.BackupBlob{}

	if d.Jobs.AnyNeedDefaultBlobsForRestore() {
		blobs = append(blobs, NewDefaultBlob(d, d.SSHConnection, d.Logger))
	}

	for _, job := range d.Jobs.WithNamedRestoreBlobs() {
		blobs = append(blobs, NewNamedRestoreBlob(d, job, d.SSHConnection, d.Logger))
	}

	return blobs
}

func (d *DeployedInstance) RunOnInstance(cmd, label string) ([]byte, []byte, int, error) {
	d.Logger.Debug("", "Running %s on %s/%s", label, d.instanceGroupName, d.instanceID)

	stdout, stderr, exitCode, err := d.Run(cmd)
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, d.instanceGroupName, d.instanceID, exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}

func (d *DeployedInstance) Name() string {
	return d.instanceGroupName
}

func (d *DeployedInstance) Index() string {
	return d.backupAndRestoreInstanceIndex
}

func (d *DeployedInstance) ID() string {
	return d.instanceID
}

func (d *DeployedInstance) runAndHandleErrs(label, jobName string, script Script) error {
	d.Logger.Debug("", "> %s", script)

	stdout, stderr, exitCode, err := d.RunOnInstance(
		fmt.Sprintf(
			"sudo %s",
			script,
		),
		label,
	)

	return d.handleErrs(jobName, label, err, exitCode, stdout, stderr)
}

func (d *DeployedInstance) handleErrs(jobName, label string, err error, exitCode int, stdout, stderr []byte) error {
	var foundErrors []error

	if err != nil {
		d.Logger.Error("", fmt.Sprintf(
			"Error attempting to run %s script for job %s on %s/%s. Error: %s",
			label,
			jobName,
			d.instanceGroupName,
			d.instanceID,
			err.Error(),
		))
		foundErrors = append(foundErrors, err)
	}

	if exitCode != 0 {
		errorString := fmt.Sprintf(
			"%s script for job %s failed on %s/%s.\nStdout: %s\nStderr: %s",
			label,
			jobName,
			d.instanceGroupName,
			d.instanceID,
			stdout,
			stderr,
		)

		foundErrors = append(foundErrors, errors.New(errorString))

		d.Logger.Error("", errorString)
	}

	return orchestrator.ConvertErrors(foundErrors)
}
