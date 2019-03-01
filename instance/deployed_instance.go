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
	Logger
	jobs         orchestrator.Jobs
	remoteRunner ssh.RemoteRunner
}

func NewDeployedInstance(instanceIndex string, instanceGroupName string, instanceID string, artifactDirCreated bool, remoteRunner ssh.RemoteRunner, logger Logger, jobs orchestrator.Jobs) *DeployedInstance {
	return &DeployedInstance{
		backupAndRestoreInstanceIndex: instanceIndex,
		instanceGroupName:             instanceGroupName,
		instanceID:                    instanceID,
		artifactDirCreated:            artifactDirCreated,
		Logger:                        logger,
		jobs:                          jobs,
		remoteRunner:                  remoteRunner,
	}
}

func (i *DeployedInstance) ArtifactDirExists() (bool, error) {
	return i.remoteRunner.DirectoryExists(orchestrator.ArtifactDirectory)
}

func (i *DeployedInstance) RemoveArtifactDir() error {
	return i.remoteRunner.RemoveDirectory(orchestrator.ArtifactDirectory)
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

func (i *DeployedInstance) HasMetadataRestoreNames() bool {
	return i.jobs.HasMetadataRestoreNames()
}

func (i *DeployedInstance) Jobs() []orchestrator.Job {
	return i.jobs
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

func artifactDirectoryVariables(artifactDirectory string) map[string]string {
	return map[string]string{
		"BBR_ARTIFACT_DIRECTORY": artifactDirectory + "/",
		"ARTIFACT_DIRECTORY":     artifactDirectory + "/",
	}
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

func (i *DeployedInstance) IsRestorable() bool {
	return i.jobs.AnyAreRestorable()
}

func (i *DeployedInstance) ArtifactsToBackup() []orchestrator.BackupArtifact {
	artifacts := []orchestrator.BackupArtifact{}

	for _, job := range i.jobs.Backupable() {
		artifacts = append(artifacts, NewBackupArtifact(job, i, i.remoteRunner, i.Logger))
	}

	return artifacts
}

func (i *DeployedInstance) ArtifactsToRestore() []orchestrator.BackupArtifact {
	artifacts := []orchestrator.BackupArtifact{}

	for _, job := range i.jobs.Restorable() {
		artifacts = append(artifacts, NewRestoreArtifact(job, i, i.remoteRunner, i.Logger))
	}

	return artifacts
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

func (i *DeployedInstance) ConnectedUsername() string {
	return i.remoteRunner.ConnectedUsername()
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
