package resourcemanagers

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/sproto"
	cproto "github.com/determined-ai/determined/master/pkg/container"
)

// container tracks an actual task container running in the cluster.
type container struct {
	req   *sproto.AllocateRequest
	id    cproto.ID
	slots int
	agent *agentState
}

// newContainer returns a new container state assigned to the specified agent.
func newContainer(req *sproto.AllocateRequest, agent *agentState, slots int) *container {
	return &container{
		req:   req,
		id:    cproto.ID(uuid.New().String()),
		slots: slots,
		agent: agent,
	}
}
