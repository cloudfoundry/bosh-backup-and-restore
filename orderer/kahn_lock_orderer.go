package orderer

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

type KahnLockOrderer struct{}

func NewKahnLockOrderer() KahnLockOrderer {
	return KahnLockOrderer{}
}

type lockingDependency struct {
	Before *orchestrator.Job
	After  *orchestrator.Job
}

func (lo KahnLockOrderer) Order(jobs []orchestrator.Job) []orchestrator.Job {
	var jobPointers = pointerify(jobs)
	var lockingDependencies = FindLockingDependencies(jobPointers)
	var orderedJobPointers = OrderJobsUsingTheKahnAlgorithm(jobPointers, lockingDependencies)
	return unpointerify(orderedJobPointers)
}

func pointerify(jobs []orchestrator.Job) []*orchestrator.Job {
	var pointers []*orchestrator.Job
	for _, job := range jobs {
		var currentJob = job
		pointers = append(pointers, &currentJob)
	}
	return pointers
}

func unpointerify(jobPointers []*orchestrator.Job) []orchestrator.Job {
	jobs := []orchestrator.Job{}
	for _, jobPointer := range jobPointers {
		jobs = append(jobs, *jobPointer)
	}
	return jobs
}
