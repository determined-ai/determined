package resourcemanagers

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// Scheduler assigns pending tasks to agents (depending on cluster availability) or requests
// running tasks to terminate.
type Scheduler interface {
	Schedule(rp *ResourcePool) ([]*AllocateRequest, []*actor.Ref)
}

// MakeScheduler returns the corresponding scheduler implementation.
func MakeScheduler(schedulingPolicy string) Scheduler {
	switch schedulingPolicy {
	case "priority":
		return NewPriorityScheduler()
	case "fair_share":
		return NewFairShareScheduler()
	default:
		panic(fmt.Sprintf("invalid scheduler: %s", schedulingPolicy))
	}
}
