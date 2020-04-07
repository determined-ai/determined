package scheduler

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	cproto "github.com/determined-ai/determined/master/pkg/container"
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
		ContainerID *cproto.ID
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
	// StartTask notifies the agent to start the task with the provided task spec.
	StartTask struct {
		Task *actor.Ref
		agent.StartContainer
	}
)

// Outgoing agent actor messages; agent actors must send these events back to the cluster.
type (
	// ContainerStartedOnAgent notifies the cluster that the task container has started running.
	ContainerStartedOnAgent struct {
		ContainerID ContainerID
		Addresses   []Address
	}
	// ContainerTerminatedOnAgent notifies the cluster that the task container has been terminated
	// with the provided reason.
	ContainerTerminatedOnAgent struct {
		ContainerID ContainerID
		Reason      agent.ContainerStopped
	}
)

// agentState holds the scheduler state for an agent. The implementation of agent-related operations
// (e.g., socket I/O) is deferred to the actor.
type agentState struct {
	handler    *actor.Ref
	devices    map[device.Device]*cproto.ID
	containers map[ContainerID]*container
	label      string
}

// newAgentState returns a new agent empty agent state backed by the handler.
func newAgentState(msg AddAgent) *agentState {
	return &agentState{
		handler:    msg.Agent,
		label:      msg.Label,
		devices:    make(map[device.Device]*cproto.ID),
		containers: make(map[ContainerID]*container),
	}
}

func (a *agentState) numSlots() int {
	return len(a.devices)
}

// numEmptySlots returns the number of slots that have not been allocated to containers.
func (a *agentState) numEmptySlots() (slots int) {
	return a.numSlots() - a.numUsedSlots()
}

// numUsedSlots returns the number of slots that have been allocated to containers.
func (a *agentState) numUsedSlots() (slots int) {
	for _, id := range a.devices {
		if id != nil {
			slots++
		}
	}
	return slots
}

func (a *agentState) assignFreeDevices(slots int, id ContainerID) []device.Device {
	if slots == 0 {
		return nil
	}
	cid := cproto.ID(id)
	devices := make([]device.Device, 0, slots)
	for d, dcid := range a.devices {
		if dcid == nil {
			a.devices[d] = &cid
			devices = append(devices, d)
		}
		if len(devices) == slots {
			break
		}
	}
	check.Panic(check.True(len(devices) == slots, "not enough devices"))
	return devices
}
