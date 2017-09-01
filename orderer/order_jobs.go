package orderer

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

func OrderJobsUsingTheKahnAlgorithm(jobs []orchestrator.Job, lockingDependencies []lockingDependency) []orchestrator.Job {
	orderedJobs := []orchestrator.Job{}

	for len(jobs) != 0 {
		jobsToLock := jobsThatCanBeLocked(jobs, lockingDependencies)
		jobs = removeJobs(jobs, jobsToLock)
		lockingDependencies = removeDependenciesThatHaveAnyOneJobInBefore(lockingDependencies, jobsToLock)

		orderedJobs = append(orderedJobs, jobsToLock...)
	}

	return orderedJobs
}

func jobsThatCanBeLocked(jobs []orchestrator.Job, dependencies []lockingDependency) []orchestrator.Job {
	var jobsWithNoDeps []orchestrator.Job
	for _, job := range jobs {
		var dependencyFound bool
		for _, dependency := range dependencies {
			if AreTheSameJob(dependency.After, job) {
				dependencyFound = true
			}
		}
		if !dependencyFound {
			jobsWithNoDeps = append(jobsWithNoDeps, job)
		}
	}
	return jobsWithNoDeps
}

func removeJobs(jobs []orchestrator.Job, jobsToRemove []orchestrator.Job) []orchestrator.Job {
	var jobsToKeep []orchestrator.Job
	for _, job := range jobs {
		var removeJob bool
		for _, jobToRemove := range jobsToRemove {
			if AreTheSameJob(jobToRemove, job) {
				removeJob = true
			}
		}

		if !removeJob {
			jobsToKeep = append(jobsToKeep, job)
		}
	}
	return jobsToKeep
}

func removeDependenciesThatHaveAnyOneJobInBefore(dependencies []lockingDependency, jobs []orchestrator.Job) []lockingDependency {
	var dependenciesToKeep []lockingDependency

	for _, dependency := range dependencies {
		var removeDep bool
		for _, job := range jobs {
			if AreTheSameJob(dependency.Before, job) {
				removeDep = true
			}
		}

		if !removeDep {
			dependenciesToKeep = append(dependenciesToKeep, dependency)
		}
	}

	return dependenciesToKeep
}

func AreTheSameJob(left, right orchestrator.Job) bool {
	return left.Name() == right.Name() && left.InstanceIdentifier() == right.InstanceIdentifier()
}
