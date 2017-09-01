package orderer

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

func FindLockingDependencies(jobs []orchestrator.Job) []lockingDependency {
	var lockingDependencies []lockingDependency

	for _, job := range jobs {
		jobSpecifiersThatShouldBeLockedAfter := job.ShouldBeLockedBefore()

		for _, jobSpecifierThatShouldBeLockedAfter := range jobSpecifiersThatShouldBeLockedAfter {
			jobsThatShouldBeLockedAfter := findJobsBySpecifier(jobs, jobSpecifierThatShouldBeLockedAfter)
			for _, afterJob := range jobsThatShouldBeLockedAfter {
				lockingDependencies = append(lockingDependencies, lockingDependency{Before: job, After: afterJob})
			}
		}
	}

	return lockingDependencies
}

func findJobsBySpecifier(jobs []orchestrator.Job, specifier orchestrator.JobSpecifier) []orchestrator.Job {
	var foundJobs []orchestrator.Job
	for _, job := range jobs {
		if job.Name() == specifier.Name && job.Release() == specifier.Release {
			foundJobs = append(foundJobs, job)
		}
	}
	return foundJobs
}
