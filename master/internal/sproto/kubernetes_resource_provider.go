package sproto

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// Incoming pods actor messages; pods actors must accept these messages.
type (
	// StartPod notifies the pods actor to start a pod with the task spec.
	StartPod struct {
		TaskHandler *actor.Ref
		Spec        tasks.TaskSpec
		Slots       int
		Rank        int
	}
)
