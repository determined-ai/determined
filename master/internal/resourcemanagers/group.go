package resourcemanagers

import "github.com/determined-ai/determined/master/pkg/actor"

// groupActorStopped notifies that the group actor is stopped.
type groupActorStopped struct {
	Ref *actor.Ref
}

// group manages the common state for a set of tasks that all share the same scheduling restrictions
// (e.g. max slots or fair share weight).
type group struct {
	handler  *actor.Ref
	maxSlots *int
	weight   float64
}
