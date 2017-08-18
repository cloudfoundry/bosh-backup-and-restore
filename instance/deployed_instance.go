package instance

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

type DeployedInstance struct {
	backupAndRestoreInstanceIndex string
	instanceID                    string
	instanceGroupName             string
	artifactDirCreated            bool
	ssh.SSHConnection
	Logger
	jobs orchestrator.Jobs
}

func NewDeployedInstance(instanceIndex string, instanceGroupName string, instanceID string, artifactDirCreated bool, connection ssh.SSHConnection, logger Logger, jobs orchestrator.Jobs) *DeployedInstance {
	deployedInstance := &DeployedInstance{
		backupAndRestoreInstanceIndex: instanceIndex,
		instanceGroupName:             instanceGroupName,
		instanceID:                    instanceID,
		artifactDirCreated:            artifactDirCreated,
		SSHConnection:                 connection,
		Logger:                        logger,
		jobs:                          jobs,
	}
	return deployedInstance
}

func (i *DeployedInstance) ArtifactDirExists() (bool, error) {
	_, _, exitCode, err := i.RunOnInstance(
		fmt.Sprintf(
			"stat %s",
			orchestrator.ArtifactDirectory,
		),
		"artifact directory check",
	)

	return exitCode == 0, err
}

func (i *DeployedInstance) IsBackupable() bool {
	return i.jobs.AnyAreBackupable()
}

func (i *DeployedInstance) ArtifactDirCreated() bool {
	return i.artifactDirCreated
}

func (i *DeployedInstance) MarkArtifactDirCreated() {
	i.artifactDirCreated = true
}

func (i *DeployedInstance) CustomBackupArtifactNames() []string {
	return i.jobs.CustomBackupArtifactNames()
}

func (i *DeployedInstance) CustomRestoreArtifactNames() []string {
	return i.jobs.CustomRestoreArtifactNames()
}

func (i *DeployedInstance) Jobs() []orchestrator.Job {
	return i.jobs
}

func (i *DeployedInstance) PreBackupLock() error {
	var preBackupLockErrors []error
	for _, job := range i.jobs {
		if err := job.PreBackupLock(); err != nil {
			preBackupLockErrors = append(preBackupLockErrors, err)
		}
	}

	return orchestrator.ConvertErrors(preBackupLockErrors)
}

func (i *DeployedInstance) Backup() error {
	var backupErrors []error
	for _, job := range i.jobs {
		if err := job.Backup(); err != nil {
			backupErrors = append(backupErrors, err)
		}
	}

	if i.IsBackupable() {
		i.artifactDirCreated = true
	}

	return orchestrator.ConvertErrors(backupErrors)
}

func artifactDirectoryVariables(artifactDirectory string) string {
	return fmt.Sprintf("BBR_ARTIFACT_DIRECTORY=%s/ ARTIFACT_DIRECTORY=%[1]s/", artifactDirectory)
}

func (i *DeployedInstance) PostBackupUnlock() error {
	var unlockErrors []error
	for _, job := range i.jobs {
		if err := job.PostBackupUnlock(); err != nil {
			unlockErrors = append(unlockErrors, err)
		}
	}

	return orchestrator.ConvertErrors(unlockErrors)
}

func (i *DeployedInstance) Restore() error {
	var restoreErrors []error
	for _, job := range i.jobs {
		if err := job.Restore(); err != nil {
			restoreErrors = append(restoreErrors, err)
		}
	}

	return orchestrator.ConvertErrors(restoreErrors)
}

func (i *DeployedInstance) PostRestoreUnlock() error {
	var unlockErrors []error
	for _, job := range i.jobs {
		if err := job.PostRestoreUnlock(); err != nil {
			unlockErrors = append(unlockErrors, err)
		}
	}

	return orchestrator.ConvertErrors(unlockErrors)
}

func (i *DeployedInstance) IsRestorable() bool {
	return i.jobs.AnyAreRestorable()
}

func (i *DeployedInstance) ArtifactsToBackup() []orchestrator.BackupArtifact {
	artifacts := []orchestrator.BackupArtifact{}

	for _, job := range i.jobs {
		artifacts = append(artifacts, NewBackupArtifact(job, i, i.SSHConnection, i.Logger))
	}

	return artifacts
}

func (i *DeployedInstance) ArtifactsToRestore() []orchestrator.BackupArtifact {
	artifacts := []orchestrator.BackupArtifact{}

	for _, job := range i.jobs {
		artifacts = append(artifacts, NewRestoreArtifact(job, i, i.SSHConnection, i.Logger))
	}

	return artifacts
}

func (i *DeployedInstance) RunOnInstance(cmd, label string) ([]byte, []byte, int, error) {
	i.Logger.Debug("bbr", "Running %s on %s/%s", label, i.instanceGroupName, i.instanceID)

	stdout, stderr, exitCode, err := i.Run(cmd)
	i.Logger.Debug("bbr", "Stdout: %s", string(stdout))
	i.Logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		i.Logger.Debug("bbr", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, i.instanceGroupName, i.instanceID, exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}

func (i *DeployedInstance) Name() string {
	return i.instanceGroupName
}

func (i *DeployedInstance) Index() string {
	return i.backupAndRestoreInstanceIndex
}

func (i *DeployedInstance) ID() string {
	return i.instanceID
}

func (i *DeployedInstance) handleErrs(jobName, label string, err error, exitCode int, stdout, stderr []byte) error {
	var foundErrors []error

	if err != nil {
		i.Logger.Error("bbr", fmt.Sprintf(
			"Error attempting to run %s script for job %s on %s/%s. Error: %s",
			label,
			jobName,
			i.instanceGroupName,
			i.instanceID,
			err.Error(),
		))
		foundErrors = append(foundErrors, err)
	}

	if exitCode != 0 {
		errorString := fmt.Sprintf(
			"%s script for job %s failed on %s/%s.\nStdout: %s\nStderr: %s",
			label,
			jobName,
			i.instanceGroupName,
			i.instanceID,
			stdout,
			stderr,
		)

		foundErrors = append(foundErrors, errors.New(errorString))

		i.Logger.Error("bbr", errorString)
	}

	return orchestrator.ConvertErrors(foundErrors)
}
