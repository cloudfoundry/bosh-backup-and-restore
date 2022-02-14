package instance

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

type InstanceIdentifier struct {
	InstanceGroupName string
	InstanceId        string
	Bootstrap         bool
}

func (i InstanceIdentifier) String() string {
	return fmt.Sprintf("%s/%s", i.InstanceGroupName, i.InstanceId)
}

//go:generate counterfeiter -o fakes/fake_job_finder.go . JobFinder
type JobFinder interface {
	FindJobs(instanceIdentifier InstanceIdentifier, remoteRunner ssh.RemoteRunner, manifestQuerier ManifestQuerier) (orchestrator.Jobs, error)
}

type JobFinderFromScripts struct {
	bbrVersion       string
	Logger           Logger
	parseJobMetadata MetadataParserFunc
}

func NewJobFinder(bbrVersion string, logger Logger) *JobFinderFromScripts {
	return &JobFinderFromScripts{
		bbrVersion:       bbrVersion,
		Logger:           logger,
		parseJobMetadata: ParseJobMetadata,
	}
}

func NewJobFinderOmitMetadataReleases(bbrVersion string, logger Logger) *JobFinderFromScripts {
	return &JobFinderFromScripts{
		bbrVersion:       bbrVersion,
		Logger:           logger,
		parseJobMetadata: ParseJobMetadataOmitReleases,
	}
}

func (j *JobFinderFromScripts) FindJobs(instanceIdentifier InstanceIdentifier, remoteRunner ssh.RemoteRunner,
	manifestQuerier ManifestQuerier) (orchestrator.Jobs, error) {

	findOutput, err := j.findBBRScripts(instanceIdentifier, remoteRunner)
	if err != nil {
		return nil, err
	}
	metadata := map[string]Metadata{}
	scripts := NewBackupAndRestoreScripts(findOutput)
	for _, script := range scripts {
		if script.isMetadata() {
			jobMetadata, err := j.findMetadata(instanceIdentifier, script, remoteRunner)

			if err != nil {
				return nil, err
			}

			jobName := script.JobName()
			j.logMetadata(jobMetadata, jobName)

			jobMetadata.BackupName = ""
			metadata[jobName] = *jobMetadata
		}
	}

	return j.buildJobs(remoteRunner, instanceIdentifier, j.Logger, scripts, metadata, manifestQuerier)

}

func (j *JobFinderFromScripts) logMetadata(jobMetadata *Metadata, jobName string) {
	for _, lockBefore := range jobMetadata.BackupShouldBeLockedBefore {
		j.Logger.Info("bbr", "Detected order: %s should be locked before %s during backup", jobName, filepath.Join(lockBefore.Release, lockBefore.JobName))
	}
	for _, lockBefore := range jobMetadata.RestoreShouldBeLockedBefore {
		j.Logger.Info("bbr", "Detected order: %s should be locked before %s during restore", jobName, filepath.Join(lockBefore.Release, lockBefore.JobName))
	}

	if jobMetadata.BackupName != "" {
		j.Logger.Warn("bbr", "discontinued metadata keys backup_name/restore_name found in job %s. bbr will not be able to restore this backup artifact.", jobName)
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
	metadataContent, err := remoteRunner.RunScriptWithEnv(
		string(script),
		map[string]string{"BBR_VERSION": j.bbrVersion},
		fmt.Sprintf("find metadata for %s on %s", script.JobName(), instanceIdentifier),
		io.Discard,
	)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"An error occurred while running metadata script for job %s on %s",
			script.JobName(),
			instanceIdentifier,
		)
	}
	jobMetadata, err := j.parseJobMetadata(metadataContent)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"Parsing metadata from job %s on %s failed",
			script.JobName(),
			instanceIdentifier,
		)
	}

	return jobMetadata, nil
}

func (j *JobFinderFromScripts) buildJobs(remoteRunner ssh.RemoteRunner,
	instanceIdentifier InstanceIdentifier,
	logger Logger, scripts BackupAndRestoreScripts,
	metadata map[string]Metadata, manifestQuerier ManifestQuerier) (orchestrator.Jobs, error) {
	groupedByJobName := map[string]BackupAndRestoreScripts{}
	for _, script := range scripts {
		jobName := script.JobName()
		existingScripts := groupedByJobName[jobName]
		groupedByJobName[jobName] = append(existingScripts, script)
	}
	var jobs orchestrator.Jobs
	var skippedJobs []string
	for jobName, jobScripts := range groupedByJobName {
		if metadata[jobName].SkipBBRScripts {
			skippedJobs = append(skippedJobs, jobName)
			continue
		}

		for _, jobScript := range jobScripts {
			j.Logger.Info("bbr", "%s/%s/%s", instanceIdentifier, jobName, jobScript.Name())
		}

		releaseName, err := manifestQuerier.FindReleaseName(instanceIdentifier.InstanceGroupName, jobName)
		if err != nil {
			logger.Warn("bbr", "could not find release name for job %s", jobName)
			releaseName = ""
		}

		backupOneRestoreAll, _ := manifestQuerier.IsJobBackupOneRestoreAll(instanceIdentifier.InstanceGroupName, jobName)

		jobs = append(jobs, NewJob(
			remoteRunner,
			instanceIdentifier.String(),
			logger,
			releaseName,
			jobScripts,
			metadata[jobName],
			backupOneRestoreAll,
			instanceIdentifier.Bootstrap,
		))
	}

	var skippedJobsMsg = "Found disabled jobs on instance"
	if len(skippedJobs) != 0 {
		skippedJobsMsg = fmt.Sprintf("%s %s jobs:", skippedJobsMsg, instanceIdentifier)
		for _, job := range skippedJobs {
			skippedJobsMsg = skippedJobsMsg + " " + job
		}
		j.Logger.Debug("bbr", skippedJobsMsg)
	}
	return jobs, nil
}
