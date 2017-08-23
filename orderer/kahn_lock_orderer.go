package orderer

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

type KahnLockOrderer struct{}

func NewKahnLockOrderer() KahnLockOrderer {
	return KahnLockOrderer{}
}

func (lo KahnLockOrderer) Order(jobs []orchestrator.Job) []orchestrator.Job {
	var lockingDependencies = newDependencySet()
	for _, job := range jobs {
		lockingDependencies.Add(lockingDependenciesFor(job)...)
	}

	var jobSpecifiers = newJobSpecifierSet()
	for _, job := range jobs {
		jobSpecifiers.Add(jobSpecifierFor(job))
	}

	splitOutFreeDependencies(jobSpecifiers, lockingDependencies)

	return []orchestrator.Job{}
}

func splitOutFreeDependencies(jobs jobSpecifierSet, deps dependencySet) []orchestrator.JobSpecifier {
	var jobsWithSatisfiedDependencies = []orchestrator.JobSpecifier{}
	var remainingJobs = newJobSpecifierSet()
	for jobSpecifier := range jobs {
		if deps.AreAllSatisfiedFor(jobSpecifier) {
			jobsWithSatisfiedDependencies = append(jobsWithSatisfiedDependencies, jobSpecifier)
		} else {
			remainingJobs.Add(jobSpecifier)
		}
	}

	if len(jobsWithSatisfiedDependencies) == 0 {
		if len(deps) == 0 {
			return []orchestrator.JobSpecifier{}
		} else {
			panic("loop!")
		}
	} else {
		for _, jobSpecifier := range jobsWithSatisfiedDependencies {
			deps.RemoveDependenciesOn(jobSpecifier)
		}
		return append(jobsWithSatisfiedDependencies, splitOutFreeDependencies(remainingJobs, deps)...)
	}
}

func jobSpecifierFor(job orchestrator.Job) orchestrator.JobSpecifier {
	return orchestrator.JobSpecifier{Name: job.Name()}
}
func lockingDependenciesFor(j orchestrator.Job) []lockingDependency {
	dependencies := []lockingDependency{}
	for _, lockBeforeSpecifier := range j.ShouldBeLockedBefore() {
		newDependency := NewLockingDependency(jobSpecifierFor(j), lockBeforeSpecifier)
		dependencies = append(dependencies, *newDependency)
	}
	return dependencies
}

func NewLockingDependency(dependee, dependent orchestrator.JobSpecifier) *lockingDependency {
	return &lockingDependency{
		ShouldRunFirst: dependee,
		DependentJob:   dependent,
	}
}

type lockingDependency struct {
	ShouldRunFirst orchestrator.JobSpecifier
	DependentJob   orchestrator.JobSpecifier
}

type jobSpecifierSet map[orchestrator.JobSpecifier]bool

func newJobSpecifierSet() jobSpecifierSet {
	return make(map[orchestrator.JobSpecifier]bool)
}

func (set jobSpecifierSet) Add(jobSpecifier orchestrator.JobSpecifier) {
	set[jobSpecifier] = true
}

func (set jobSpecifierSet) Delete(jobSpecifier orchestrator.JobSpecifier) {
	delete(set, jobSpecifier)
}

type dependencySet map[lockingDependency]bool

func newDependencySet() dependencySet {
	return make(map[lockingDependency]bool)
}

func (deps dependencySet) Add(dependencies ...lockingDependency) {
	for _, dependency := range dependencies {
		deps[dependency] = true
	}
}

func (deps dependencySet) AreAllSatisfiedFor(specifier orchestrator.JobSpecifier) bool {
	for dependency := range deps {
		if dependency.DependentJob == specifier {
			return false
		}
	}
	return true
}

func (deps dependencySet) RemoveDependenciesOn(specifier orchestrator.JobSpecifier) {
	for dependency := range deps {
		if dependency.ShouldRunFirst == specifier {
			delete(deps, dependency)
		}
	}
}

// function that takes a list of jobs, and a list of jobOrderPairings, and returns
// 1. a list of jobs we can run now
// 2. a reduced list of orderings
