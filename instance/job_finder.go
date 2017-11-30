package instance

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
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
	FindJobs(instanceIdentifier InstanceIdentifier, remoteRunner ssh.RemoteRunner, releaseMapping ReleaseMapping) (orchestrator.Jobs, error)
}

type JobFinderFromScripts struct {
	Logger Logger
}

func NewJobFinder(logger Logger) *JobFinderFromScripts {
	return &JobFinderFromScripts{
		Logger: logger,
	}
}

func (j *JobFinderFromScripts) FindJobs(instanceIdentifier InstanceIdentifier, remoteRunner ssh.RemoteRunner,
	releaseMapping ReleaseMapping) (orchestrator.Jobs, error) {
	findOutput, err := j.findBBRScripts(instanceIdentifier, remoteRunner)
	if err != nil {
		return nil, err
	}
	metadata := map[string]Metadata{}
	scripts := NewBackupAndRestoreScripts(findOutput)
	for _, script := range scripts {
		j.Logger.Info("bbr", "%s/%s/%s", instanceIdentifier, script.JobName(), script.Name())
		if script.isMetadata() {
			jobMetadata, err := j.findMetadata(instanceIdentifier, script, remoteRunner)

			if err != nil {
				return nil, err
			}

			jobName := script.JobName()
			metadata[jobName] = *jobMetadata
			j.logMetadata(jobMetadata, script.JobName())
		}
	}

	return j.buildJobs(remoteRunner, instanceIdentifier, j.Logger, scripts, metadata, releaseMapping)
}

func (j *JobFinderFromScripts) logMetadata(jobMetadata *Metadata, jobName string) {
	for _, lockBefore := range jobMetadata.BackupShouldBeLockedBefore {
		j.Logger.Info("bbr", "Detected order: %s should be locked before %s/%s during backup", jobName, lockBefore.Release, lockBefore.JobName)
	}
	for _, lockBefore := range jobMetadata.RestoreShouldBeLockedBefore {
		j.Logger.Info("bbr", "Detected order: %s should be locked before %s/%s during restore", jobName, lockBefore.Release, lockBefore.JobName)
	}
}

func (j *JobFinderFromScripts) findBBRScripts(instanceIdentifierForLogging InstanceIdentifier,
	remoteRunner ssh.RemoteRunner) ([]string, error) {
	j.Logger.Debug("bbr", "Attempting to find scripts on %s", instanceIdentifierForLogging)

	scripts, err := remoteRunner.FindFiles("/var/vcap/jobs/*/bin/bbr/*")
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("finding scripts failed on %s", instanceIdentifierForLogging))
	}

	return scripts, nil
}

func (j *JobFinderFromScripts) findMetadata(instanceIdentifier InstanceIdentifier, script Script, remoteRunner ssh.RemoteRunner) (*Metadata, error) {
	metadataContent, err := remoteRunner.RunScript(string(script))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(
			"An error occurred while running metadata script for job %s on %s: %s",
			script.JobName(),
			instanceIdentifier,
			err,
		))
	}

	jobMetadata, err := ParseJobMetadata(metadataContent)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(
			"Parsing metadata from job %s on %s failed: %s",
			script.JobName(),
			instanceIdentifier,
			err.Error(),
		))
	}

	return jobMetadata, nil
}

func (j *JobFinderFromScripts) buildJobs(remoteRunner ssh.RemoteRunner,
	instanceIdentifier InstanceIdentifier,
	logger Logger, scripts BackupAndRestoreScripts,
	metadata map[string]Metadata, releaseMapping ReleaseMapping) (orchestrator.Jobs, error) {
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

		jobs = append(jobs, NewJob(remoteRunner, instanceIdentifier.String(), logger, releaseName, jobScripts, metadata[jobName]))
	}

	return jobs, nil
}
