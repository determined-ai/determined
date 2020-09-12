package sproto

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// Incoming pods actor messages; pods actors must accept these messages.
type (
	// StartTaskPod notifies the pods actor to start a pod with the task spec.
	StartTaskPod struct {
		TaskActor *actor.Ref
		Spec      tasks.TaskSpec
		Slots     int
	}
)
