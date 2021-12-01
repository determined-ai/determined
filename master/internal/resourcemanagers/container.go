package resourcemanagers

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
)

// container tracks an actual task container running in the cluster.
type container struct {
	req   *sproto.AllocateRequest
	id    cproto.ID
	slots int
}

// newContainer returns a new container state assigned to the specified agent.
func newContainer(req *sproto.AllocateRequest, slots int) *container {
	return &container{
		req:   req,
		id:    cproto.ID(uuid.New().String()),
		slots: slots,
	}
}
