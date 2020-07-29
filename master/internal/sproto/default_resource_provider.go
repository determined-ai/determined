package sproto

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
)

// DeviceID is the unique identifier for a device in the cluster.
type DeviceID struct {
	Agent  *actor.Ref
	Device device.Device
}

// Agent-related cluster level messages.
type (
	// AddAgent adds the agent to the cluster.
	AddAgent struct {
		Agent *actor.Ref
		Label string
	}
	// AddDevice makes the device immediately available for scheduling.
	AddDevice struct {
		DeviceID
		ContainerID *container.ID
	}
	// FreeDevice notifies the cluster that the device's container is no longer running.
	FreeDevice struct {
		DeviceID
	}
	// RemoveDevice removes the device from scheduling.
	RemoveDevice struct {
		DeviceID
	}
	// RemoveAgent removes the agent from the cluster.
	RemoveAgent struct {
		Agent *actor.Ref
	}
)

// Incoming agent actor messages; agent actors must accept these messages.
type (
	// StartTaskOnAgent notifies the agent to start the task with the provided task spec.
	StartTaskOnAgent struct {
		Task *actor.Ref
		agent.StartContainer
	}
)

// Messages from agents to the RP.
type (
	//TaskStartedOnAgent notifies the resource provider that the task started on the agent.
	TaskStartedOnAgent struct {
		ContainerID      container.ID
		ContainerStarted *agent.ContainerStarted
	}

	//TaskTerminatedOnAgent notifies the resource provider that the task has been terminated.
	TaskTerminatedOnAgent struct {
		ContainerID      container.ID
		ContainerStopped *agent.ContainerStopped
	}
)
