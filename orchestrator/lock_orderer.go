package orchestrator

type NoopLockOrderer struct{}

func NewNoopLockOrderer() NoopLockOrderer {
	return NoopLockOrderer{}
}

func (lo NoopLockOrderer) Order(jobs []Job) []Job {
	return jobs
}
