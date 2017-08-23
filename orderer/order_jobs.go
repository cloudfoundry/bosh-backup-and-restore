package orderer

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

func OrderJobsUsingTheKahnAlgorithm(jobs []*orchestrator.Job, lockingDependencies []lockingDependency) []*orchestrator.Job {
	orderedJobs := []*orchestrator.Job{}

	for len(jobs) != 0 {
		jobsToLock := jobsThatCanBeLocked(jobs, lockingDependencies)
		jobs = removeJobs(jobs, jobsToLock)
		lockingDependencies = removeDependeciesThatHaveAnyOneJobInBefore(lockingDependencies, jobsToLock)

		orderedJobs = append(orderedJobs, jobsToLock...)
	}

	return orderedJobs
}

func removeJobs(jobs []*orchestrator.Job, jobsToRemove []*orchestrator.Job) []*orchestrator.Job {
	var jobsToKeep []*orchestrator.Job
	for _, job := range jobs {
		var removeJob bool
		for _, jobToRemove := range jobsToRemove {
			if jobToRemove == job {
				removeJob = true
			}
		}

		if !removeJob {
			jobsToKeep = append(jobsToKeep, job)
		}
	}
	return jobsToKeep
}

func removeDependeciesThatHaveAnyOneJobInBefore(dependencies []lockingDependency, jobs []*orchestrator.Job) []lockingDependency {
	var dependenciesToKeep []lockingDependency

	for _, dependency := range dependencies {
		var removeDep bool
		for _, job := range jobs {
			if dependency.Before == job {
				removeDep = true
			}
		}

		if !removeDep {
			dependenciesToKeep = append(dependenciesToKeep, dependency)
		}
	}

	return dependenciesToKeep
}

func jobsThatCanBeLocked(jobs []*orchestrator.Job, dependencies []lockingDependency) []*orchestrator.Job {
	var jobsWithNoDeps []*orchestrator.Job
	for _, job := range jobs {
		var dependencyFound bool
		for _, dependency := range dependencies {
			if dependency.After == job {
				dependencyFound = true
			}
		}
		if !dependencyFound {
			jobsWithNoDeps = append(jobsWithNoDeps, job)
		}
	}
	return jobsWithNoDeps
}
