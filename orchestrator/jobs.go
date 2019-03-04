package orchestrator

type Jobs []Job

func (jobs Jobs) Backupable() Jobs {
	backupableJobs := Jobs{}
	for _, job := range jobs {
		if job.HasBackup() {
			backupableJobs = append(backupableJobs, job)
		}
	}
	return backupableJobs
}
func (jobs Jobs) AnyAreBackupable() bool {
	return !jobs.Backupable().empty()
}

func (jobs Jobs) Restorable() Jobs {
	restorableJobs := Jobs{}
	for _, job := range jobs {
		if job.HasRestore() {
			restorableJobs = append(restorableJobs, job)
		}
	}
	return restorableJobs
}

func (jobs Jobs) AnyAreRestorable() bool {
	return !jobs.Restorable().empty()
}

func (jobs Jobs) withNamedBackupArtifacts() Jobs {
	jobsWithNamedArtifacts := Jobs{}
	for _, job := range jobs {
		if job.HasNamedBackupArtifact() {
			jobsWithNamedArtifacts = append(jobsWithNamedArtifacts, job)
		}
	}
	return jobsWithNamedArtifacts
}

func (jobs Jobs) withNamedRestoreArtifacts() Jobs {
	jobsWithNamedArtifacts := Jobs{}
	for _, job := range jobs {
		if job.HasNamedRestoreArtifact() {
			jobsWithNamedArtifacts = append(jobsWithNamedArtifacts, job)
		}
	}
	return jobsWithNamedArtifacts
}

func (jobs Jobs) CustomRestoreArtifactNames() []string {
	var artifactNames []string

	for _, job := range jobs.withNamedRestoreArtifacts() {
		artifactNames = append(artifactNames, job.RestoreArtifactName())
	}

	return artifactNames
}

func (jobs Jobs) HasMetadataRestoreNames() bool {
	for _, job := range jobs {
		if job.HasMetadataRestoreName() {
			return true
		}
	}

	return false
}

func (jobs Jobs) empty() bool {
	return len(jobs) == 0
}
