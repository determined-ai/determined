package agent

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
)

// ContainerSnapshot is a database representation of `containerResources`.
type ContainerSnapshot struct {
	bun.BaseModel `bun:"table:resourcemanagers_agent_containers,alias:rmac"`
	ResourceID    sproto.ResourcesID `bun:"resource_id"`
	AgentID       aproto.ID          `bun:"agent_id"`
	ID            cproto.ID          `json:"id" bun:"container_id"`
	State         cproto.State       `json:"state" bun:"state"`
	Devices       []device.Device    `json:"devices" bun:"devices"`
}

// NewContainerSnapshot creates an instance from cproto.Container.
func NewContainerSnapshot(c *cproto.Container) ContainerSnapshot {
	return ContainerSnapshot{
		ID:      c.ID,
		State:   c.State,
		Devices: c.Devices,
	}
}
