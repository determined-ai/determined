package scheduler

import (
	"github.com/google/uuid"
)

// ContainerID is a unique ID assigned to the containers of tasks when started in the cluster.
type ContainerID string

// container tracks an actual task container running in the cluster.
type container struct {
	req   *AssignRequest
	id    ContainerID
	slots int
	agent *agentState
}

// newContainer returns a new container state assigned to the specified agent.
func newContainer(req *AssignRequest, agent *agentState, slots, ordinal int) *container {
	return &container{
		req:   req,
		id:    ContainerID(uuid.New().String()),
		slots: slots,
		agent: agent,
	}
}
