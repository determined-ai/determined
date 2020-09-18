package scheduler

import "github.com/determined-ai/determined/master/pkg/actor"

// Scheduler assigns pending tasks to agents (depending on cluster availability) or requests
// running tasks to terminate.
type Scheduler interface {
	Schedule(rp *DefaultRP) ([]*AllocateRequest, []*actor.Ref)
}
