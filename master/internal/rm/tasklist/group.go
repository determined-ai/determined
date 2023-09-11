package tasklist

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// GroupActorStopped notifies that the group actor is stopped.
type GroupActorStopped struct {
	JobID model.JobID
}

// Group manages the common state for a set of tasks that all share the same scheduling restrictions
// (e.g. max slots or fair share weight).
type Group struct {
	JobID    model.JobID
	MaxSlots *int
	Weight   float64
	Priority *int
}

// GroupPriorityChangeRegistry is a registry of callbacks available for when a group's priority
// changes.
var GroupPriorityChangeRegistry = NewRegistry[model.JobID, func(int) error]()
