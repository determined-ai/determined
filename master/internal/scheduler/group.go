package scheduler

import "github.com/determined-ai/determined/master/pkg/actor"

// Group-related cluster level messages.
// TODO: Consider explicit AddGroup messages.
type (
	// groupActorStopped notifies that the group actor is stopped.
	groupActorStopped struct {
		Ref *actor.Ref
	}
	// SetGroupMaxSlots sets the maximum number of slots that a group can consume in the cluster.
	SetGroupMaxSlots struct {
		MaxSlots *int
		Handler  *actor.Ref
	}
	// SetGroupWeight sets the weight of a group in the fair share scheduler.
	SetGroupWeight struct {
		Weight  float64
		Handler *actor.Ref
	}
)

// group manages the common state for a set of tasks that all share the same scheduling restrictions
// (e.g. max slots or fair share weight).
type group struct {
	handler  *actor.Ref
	maxSlots *int
	weight   float64
}
