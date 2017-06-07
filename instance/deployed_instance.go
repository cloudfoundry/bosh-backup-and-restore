package instance

import (
	"fmt"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

type DeployedInstance struct {
	backupAndRestoreInstanceIndex string
	instanceID                    string
	instanceGroupName             string
	artifactDirCreated            bool
	ssh.SSHConnection
	Logger
	Jobs
}

func NewDeployedInstance(instanceIndex string, instanceGroupName string, instanceID string, artifactDirCreated bool, connection ssh.SSHConnection, logger Logger, jobs Jobs) *DeployedInstance {
	deployedInstance := &DeployedInstance{
		backupAndRestoreInstanceIndex: instanceIndex,
		instanceGroupName:             instanceGroupName,
		instanceID:                    instanceID,
		artifactDirCreated:            artifactDirCreated,
		SSHConnection:                 connection,
		Logger:                        logger,
		Jobs:                          jobs,
	}
	return deployedInstance
}

func (d *DeployedInstance) ArtifactDirExists() (bool, error) {
	_, _, exitCode, err := d.RunOnInstance(
		fmt.Sprintf(
			"stat %s",
			orchestrator.ArtifactDirectory,
		),
		"artifact directory check",
	)

	return exitCode == 0, err
}

func (d *DeployedInstance) HasBackupScript() bool {
	return d.Jobs.AnyAreBackupable()
}

func (d *DeployedInstance) ArtifactDirCreated() bool {
	return d.artifactDirCreated
}

func (d *DeployedInstance) MarkArtifactDirCreated() {
	d.artifactDirCreated = true
}

func (d *DeployedInstance) IsPostBackupUnlockable() bool {
	return d.Jobs.AnyArePostBackupable()
}

func (d *DeployedInstance) IsPreBackupLockable() bool {
	return d.Jobs.AnyArePreBackupable()
}

func (d *DeployedInstance) CustomBackupArtifactNames() []string {
	return d.Jobs.CustomBackupArtifactNames()
}

func (d *DeployedInstance) CustomRestoreArtifactNames() []string {
	return d.Jobs.CustomRestoreArtifactNames()
}

func (d *DeployedInstance) PreBackupLock() error {

	var foundErrors []error

	for _, job := range d.Jobs.PreBackupable() {
		d.Logger.Info("bbr", "Locking %s on %s/%s for backup...", job.Name(), d.instanceGroupName, d.instanceID)

		if err := d.runAndHandleErrs("pre backup lock", job.Name(), job.PreBackupScript()); err != nil {
			foundErrors = append(foundErrors, err)
		}
		d.Logger.Info("bbr", "Done.")
	}

	return orchestrator.ConvertErrors(foundErrors)
}

func (d *DeployedInstance) Backup() error {

	var foundErrors []error

	for _, job := range d.Jobs.Backupable() {
		d.Logger.Debug("bbr", "> %s", job.BackupScript())
		d.Logger.Info("bbr", "Backing up %s on %s/%s...", job.Name(), d.instanceGroupName, d.instanceID)

		stdout, stderr, exitCode, err := d.RunOnInstance(
			fmt.Sprintf(
				"sudo mkdir -p %s && sudo %s %s",
				job.BackupArtifactDirectory(),
				artifactDirectoryVariables(job.BackupArtifactDirectory()),
				job.BackupScript(),
			),
			"backup",
		)

		d.artifactDirCreated = true

		if err := d.handleErrs(job.Name(), "backup", err, exitCode, stdout, stderr); err != nil {
			foundErrors = append(foundErrors, err)
		}

		d.Logger.Info("bbr", "Done.")
	}

	return orchestrator.ConvertErrors(foundErrors)
}
func artifactDirectoryVariables(artifactDirectory string) string {
	return fmt.Sprintf("BBR_ARTIFACT_DIRECTORY=%s/ ARTIFACT_DIRECTORY=%[1]s/", artifactDirectory)
}

func (d *DeployedInstance) PostBackupUnlock() error {

	var foundErrors []error

	for _, job := range d.Jobs.PostBackupable() {
		d.Logger.Info("bbr", "Unlocking %s on %s/%s...", job.Name(), d.instanceGroupName, d.instanceID)

		if err := d.runAndHandleErrs("unlock", job.Name(), job.PostBackupScript()); err != nil {
			foundErrors = append(foundErrors, err)
		}
		d.Logger.Info("bbr", "Done.")
	}

	return orchestrator.ConvertErrors(foundErrors)
}

func (d *DeployedInstance) Restore() error {
	var restoreErrors []error

	for _, job := range d.Jobs.Restorable() {
		d.Logger.Debug("bbr", "> %s", job.RestoreScript())
		d.Logger.Info("bbr", "Restoring %s on %s/%s...", job.Name(), d.instanceGroupName, d.instanceID)

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
		d.Logger.Info("bbr", "Done.")
	}

	return orchestrator.ConvertErrors(restoreErrors)
}

func (d *DeployedInstance) IsRestorable() bool {
	return d.Jobs.AnyAreRestorable()
}

func (d *DeployedInstance) ArtifactsToBackup() []orchestrator.BackupArtifact {
	artifacts := []orchestrator.BackupArtifact{}

	for _, job := range d.Jobs {
		artifacts = append(artifacts, NewBackupArtifact(job, d, d.SSHConnection, d.Logger))
	}

	return artifacts
}

func (d *DeployedInstance) ArtifactsToRestore() []orchestrator.BackupArtifact {
	artifacts := []orchestrator.BackupArtifact{}

	for _, job := range d.Jobs {
		artifacts = append(artifacts, NewRestoreArtifact(job, d, d.SSHConnection, d.Logger))
	}

	return artifacts
}

func (d *DeployedInstance) RunOnInstance(cmd, label string) ([]byte, []byte, int, error) {
	d.Logger.Debug("bbr", "Running %s on %s/%s", label, d.instanceGroupName, d.instanceID)

	stdout, stderr, exitCode, err := d.Run(cmd)
	d.Logger.Debug("bbr", "Stdout: %s", string(stdout))
	d.Logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("bbr", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, d.instanceGroupName, d.instanceID, exitCode, err.Error())
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
	d.Logger.Debug("bbr", "> %s", script)

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
		d.Logger.Error("bbr", fmt.Sprintf(
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

		d.Logger.Error("bbr", errorString)
	}

	return orchestrator.ConvertErrors(foundErrors)
}
