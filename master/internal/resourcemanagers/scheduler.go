package resourcemanagers

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

// Scheduler assigns pending tasks to agents (depending on cluster availability) or requests
// running tasks to terminate.
type Scheduler interface {
	Schedule(rp *ResourcePool) ([]*sproto.AllocateRequest, []*actor.Ref)
}

// MakeScheduler returns the corresponding scheduler implementation.
func MakeScheduler(config *SchedulerConfig) Scheduler {
	switch config.GetType() {
	case priorityScheduling:
		return NewPriorityScheduler(config)
	case fairShareScheduling:
		return NewFairShareScheduler()
	case roundRobinScheduling:
		return NewRoundRobinScheduler()
	default:
		panic(fmt.Sprintf("invalid scheduler: %s", config.GetType()))
	}
}
