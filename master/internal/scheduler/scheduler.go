package scheduler

// Scheduler assigns pending tasks to agents (depending on cluster availability) or requests
// running tasks to terminate.
type Scheduler interface {
	Schedule(rp *DefaultRP)
}
