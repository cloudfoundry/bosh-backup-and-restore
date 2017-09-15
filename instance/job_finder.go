package instance

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

type InstanceIdentifier struct {
	InstanceGroupName string
	InstanceId        string
}

func (i InstanceIdentifier) String() string {
	return fmt.Sprintf("%s/%s", i.InstanceGroupName, i.InstanceId)
}

//go:generate counterfeiter -o fakes/fake_job_finder.go . JobFinder
type JobFinder interface {
	FindJobs(instanceIdentifier InstanceIdentifier, connection SSHConnection, releaseMapping ReleaseMapping) (orchestrator.Jobs, error)
}

type JobFinderFromScripts struct {
	Logger Logger
}

func NewJobFinder(logger Logger) *JobFinderFromScripts {
	return &JobFinderFromScripts{
		Logger: logger,
	}
}

func (j *JobFinderFromScripts) FindJobs(instanceIdentifier InstanceIdentifier, connection SSHConnection, releaseMapping ReleaseMapping) (orchestrator.Jobs, error) {
	findOutput, err := j.findBBRScripts(instanceIdentifier, connection)
	if err != nil {
		return nil, err
	}
	metadata := map[string]Metadata{}
	scripts := NewBackupAndRestoreScripts(findOutput)
	for _, script := range scripts {
		j.Logger.Info("bbr", "%s/%s/%s", instanceIdentifier, script.JobName(), script.Name())
	}
	for _, script := range scripts.MetadataOnly() {
		jobMetadata, err := j.findMetadata(instanceIdentifier, script, connection)

		if err != nil {
			return nil, err
		}

		jobName := script.JobName()
		metadata[jobName] = *jobMetadata
	}

	return j.buildJobs(connection, instanceIdentifier, j.Logger, scripts, metadata, releaseMapping)
}

func (j *JobFinderFromScripts) findBBRScripts(instanceIdentifierForLogging InstanceIdentifier, sshConnection SSHConnection) ([]string, error) {
	j.Logger.Debug("bbr", "Attempting to find scripts on %s", instanceIdentifierForLogging)

	stdout, stderr, exitCode, err := sshConnection.Run("find /var/vcap/jobs/*/bin/bbr/* -type f")
	if err != nil {
		j.Logger.Error(
			"",
			"Failed to run find on %s. Error: %s\nStdout: %s\nStderr%s",
			instanceIdentifierForLogging,
			err,
			stdout,
			stderr,
		)
		return nil, err
	}

	if exitCode != 0 {
		if strings.Contains(string(stderr), "No such file or directory") {
			j.Logger.Debug(
				"",
				"Running find failed on %s.\nStdout: %s\nStderr: %s",
				instanceIdentifierForLogging,
				stdout,
				stderr,
			)
		} else {
			j.Logger.Error(
				"",
				"Running find failed on %s.\nStdout: %s\nStderr: %s",
				instanceIdentifierForLogging,
				stdout,
				stderr,
			)
			return nil, errors.Errorf(
				"Running find failed on %s.\nStdout: %s\nStderr: %s",
				instanceIdentifierForLogging,
				stdout,
				stderr,
			)
		}
	}

	return strings.Split(string(stdout), "\n"), nil
}

func (j *JobFinderFromScripts) findMetadata(instanceIdentifier InstanceIdentifier, pathToScript Script, connection SSHConnection) (*Metadata, error) {
	metadataContent, errorContent, exitStatus, err := connection.Run(string(pathToScript))

	if err != nil {
		errorString := fmt.Sprintf(
			"An error occurred while running job metadata scripts on %s: %s",
			instanceIdentifier,
			err,
		)
		j.Logger.Error("bbr", errorString)
		return nil, errors.New(errorString)
	}

	if exitStatus != 0 {
		errorString := fmt.Sprintf(
			"An error occurred while running job metadata scripts on %s: %s",
			instanceIdentifier,
			errorContent,
		)
		j.Logger.Error("bbr", errorString)
		return nil, errors.New(errorString)
	}

	jobMetadata, err := NewJobMetadata(metadataContent)

	if err != nil {
		errorString := fmt.Sprintf(
			"Reading job metadata for %s failed: %s",
			instanceIdentifier,
			err.Error(),
		)
		j.Logger.Error("bbr", errorString)
		return nil, errors.New(errorString)
	}

	return jobMetadata, nil
}

func (j *JobFinderFromScripts) buildJobs(sshConnection SSHConnection, instanceIdentifier InstanceIdentifier, logger Logger, scripts BackupAndRestoreScripts, metadata map[string]Metadata, releaseMapping ReleaseMapping) (orchestrator.Jobs, error) {
	groupedByJobName := map[string]BackupAndRestoreScripts{}
	for _, script := range scripts {
		jobName := script.JobName()
		existingScripts := groupedByJobName[jobName]
		groupedByJobName[jobName] = append(existingScripts, script)
	}
	var jobs orchestrator.Jobs

	for jobName, jobScripts := range groupedByJobName {
		releaseName, err := releaseMapping.FindReleaseName(instanceIdentifier.InstanceGroupName, jobName)
		if err != nil {
			return nil, errors.Wrap(err, "error matching job to manifest")
		}

		jobs = append(jobs, NewJob(sshConnection, instanceIdentifier.String(), logger, releaseName, jobScripts, metadata[jobName]))
	}

	return jobs, nil
}
