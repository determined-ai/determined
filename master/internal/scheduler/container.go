package scheduler

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
)

// ContainerID is a unique ID assigned to the containers of tasks when started in the cluster.
type ContainerID string

// Container represents a single running container of a task in the cluster.
type Container interface {
	// ID returns this container's unique ID.
	ID() ContainerID
	// Task returns this container's task ID.
	TaskID() TaskID
	// Slots returns the number of slots this container is consuming.
	Slots() int
	// IsLeader returns if this container should be considered the main container of a gang.
	IsLeader() bool
	// Addresses returns a list of exposed addresses for this container.
	Addresses() []Address
	// Tell sends a message to a container's task.
	Tell(actor.Message)
}

// Address represents an exposed port on a container.
type Address struct {
	// ContainerIP is the IP address from inside the container.
	ContainerIP string `json:"container_ip"`
	// ContainerPort is the port from inside the container.
	ContainerPort int `json:"container_port"`

	// HostIP is the IP address from outside the container. This can be
	// different than the ContainerIP because of network forwarding on the host
	// machine.
	HostIP string `json:"host_ip"`
	// HostPort is the IP port from outside the container. This can be different
	// than the ContainerPort because of network forwarding on the host machine.
	HostPort int `json:"host_port"`
}

// containerState represents the current status of the container.
type containerState string

const (
	// containerStarting denotes that the container has been scheduled on an agent but has not yet
	// started running. This includes the container image being pulled locally and the container
	// runtime image being built.
	containerStarting containerState = "STARTING"
	// containerRunning denotes that the container has started running.
	containerRunning containerState = "RUNNING"
	// containerTerminating denotes that the container has been notified that it should terminate.
	containerTerminating containerState = "TERMINATING"
	// containerTerminated denotes that the container has exited and its exit status has been set.
	containerTerminated containerState = "TERMINATED"
)

var containerTransitions = map[containerState]map[containerState]bool{
	containerStarting: {
		containerRunning:     true,
		containerTerminating: true,
		containerTerminated:  true,
	},
	containerRunning: {
		containerTerminating: true,
		containerTerminated:  true,
	},
	containerTerminating: {
		containerTerminating: true,
		containerTerminated:  true,
		containerRunning:     true,
	},
}

func isValidContainerStateTransition(cur, next containerState) bool {
	return containerTransitions[cur][next]
}

// container tracks an actual task container running in the cluster.
type container struct {
	task       *Task
	id         ContainerID
	slots      int
	state      containerState
	agent      *agentState
	exitStatus *aproto.ContainerStopped
	ordinal    int
	addresses  []Address
}

// newContainer returns a new container state assigned to the specified agent.
func newContainer(task *Task, agent *agentState, slots, ordinal int) *container {
	return &container{
		task:    task,
		id:      ContainerID(uuid.New().String()),
		slots:   slots,
		agent:   agent,
		state:   containerStarting,
		ordinal: ordinal,
	}
}

func (c *container) ID() ContainerID      { return c.id }
func (c *container) TaskID() TaskID       { return c.task.ID }
func (c *container) Slots() int           { return c.slots }
func (c *container) Addresses() []Address { return c.addresses }
func (c *container) IsLeader() bool       { return c.ordinal == 0 }
func (c *container) ExitStatus() aproto.ContainerStopped {
	check.Panic(check.Equal(c.state, containerTerminated,
		"cannot fetch exit status of container that has not terminated yet"))
	return *c.exitStatus
}

func (c *container) Tell(message actor.Message) {
	h := c.task.handler
	h.System().Tell(h, message)
}

func (c *container) mustTransition(next containerState) {
	if !isValidContainerStateTransition(c.state, next) {
		panic(fmt.Sprintf("invalid container transition from %v to %v", c.state, next))
	}
	c.state = next
}
