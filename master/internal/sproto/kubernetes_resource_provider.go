package sproto

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/container"

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

// Kubernetes resource provider must accept these messages.
type (
	// PodStarted notifies the RP that the pod is now running.
	PodStarted struct {
		ContainerID     container.ID
		IP              string
		Ports           []int
		NetworkProtocol string
	}

	// PodTerminated notifies the RP that the pod is not stopped.
	PodTerminated struct {
		ContainerID      container.ID
		ContainerStopped *agent.ContainerStopped
	}
)
