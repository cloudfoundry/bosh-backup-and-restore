package orderer

import (
	"errors"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

type KahnLockOrderer struct {
	orderConstraintSpecifier orderConstraintSpecifier
}

func newKahnLockOrderer(specifier orderConstraintSpecifier) KahnLockOrderer {
	return KahnLockOrderer{
		orderConstraintSpecifier: specifier,
	}
}

func NewKahnBackupLockOrderer() KahnLockOrderer {
	return newKahnLockOrderer(NewBackupOrderConstraintSpecifier())
}

func NewKahnRestoreLockOrderer() KahnLockOrderer {
	return newKahnLockOrderer(NewRestoreOrderConstraintSpecifier())
}

type orderConstraintSpecifier interface {
	Before(job orchestrator.Job) []orchestrator.JobSpecifier
}

type lockingDependency struct {
	Before orchestrator.Job
	After  orchestrator.Job
}

func (lo KahnLockOrderer) Order(jobs []orchestrator.Job) ([]orchestrator.Job, error) {
	var lockingDependencies = findLockingDependencies(jobs, lo.orderConstraintSpecifier)
	return orderJobsUsingTheKahnAlgorithm(jobs, lockingDependencies)
}

func findLockingDependencies(jobs []orchestrator.Job, orderConstraintSpecifier orderConstraintSpecifier) []lockingDependency {
	var lockingDependencies []lockingDependency

	for _, job := range jobs {
		jobSpecifiersThatShouldBeLockedAfter := orderConstraintSpecifier.Before(job)

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

func orderJobsUsingTheKahnAlgorithm(jobs []orchestrator.Job, lockingDependencies []lockingDependency) ([]orchestrator.Job, error) {
	orderedJobs := []orchestrator.Job{}

	for len(jobs) != 0 {
		jobsToLock := jobsThatCanBeLocked(jobs, lockingDependencies)
		jobs = removeJobs(jobs, jobsToLock)
		lockingDependencies = removeDependenciesThatHaveAnyOneJobInBefore(lockingDependencies, jobsToLock)

		if len(jobsToLock) == 0 {
			return nil, errors.New("job locking dependency graph is cyclic")
		}

		orderedJobs = append(orderedJobs, jobsToLock...)
	}

	return orderedJobs, nil
}

func jobsThatCanBeLocked(jobs []orchestrator.Job, dependencies []lockingDependency) []orchestrator.Job {
	var jobsWithNoDeps []orchestrator.Job
	for _, job := range jobs {
		var dependencyFound bool
		for _, dependency := range dependencies {
			if areTheSameJob(dependency.After, job) {
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
			if areTheSameJob(jobToRemove, job) {
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
			if areTheSameJob(dependency.Before, job) {
				removeDep = true
			}
		}

		if !removeDep {
			dependenciesToKeep = append(dependenciesToKeep, dependency)
		}
	}

	return dependenciesToKeep
}

func areTheSameJob(left, right orchestrator.Job) bool {
	return left.Name() == right.Name() && left.InstanceIdentifier() == right.InstanceIdentifier()
}
