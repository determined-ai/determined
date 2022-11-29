package tasklist

import (
	"github.com/determined-ai/determined/master/pkg/actor"
)

// GroupActorStopped notifies that the group actor is stopped.
type GroupActorStopped struct {
	Ref *actor.Ref
}

// Group manages the common state for a set of tasks that all share the same scheduling restrictions
// (e.g. max slots or fair share weight).
type Group struct {
	Handler  *actor.Ref
	MaxSlots *int
	Weight   float64
	Priority *int
}
