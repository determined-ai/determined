package scheduler

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// ContainerID is a unique ID assigned to the containers of tasks when started in the cluster.
type ContainerID string

// Container represents a single running container of a task in the cluster.
type Container interface {
	// ID returns this container's unique ID.
	ID() ContainerID
	// Task returns this container's task ID.
	TaskID() RequestID
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

// container tracks an actual task container running in the cluster.
type container struct {
	req       *AssignRequest
	id        ContainerID
	slots     int
	agent     *agentState
	ordinal   int
	addresses []Address
}

// newContainer returns a new container state assigned to the specified agent.
func newContainer(req *AssignRequest, agent *agentState, slots, ordinal int) *container {
	return &container{
		req:     req,
		id:      ContainerID(uuid.New().String()),
		slots:   slots,
		agent:   agent,
		ordinal: ordinal,
	}
}

func (c *container) ID() ContainerID      { return c.id }
func (c *container) TaskID() RequestID    { return c.req.ID }
func (c *container) Slots() int           { return c.slots }
func (c *container) Addresses() []Address { return c.addresses }
func (c *container) IsLeader() bool       { return c.ordinal == 0 }

func (c *container) Tell(message actor.Message) {
	h := c.req.Handler
	h.System().Tell(h, message)
}
