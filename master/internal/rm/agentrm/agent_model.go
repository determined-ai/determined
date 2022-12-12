package agentrm

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
)

// slotData is a database representation of slot state.
type slotData struct {
	Device      device.Device
	UserEnabled bool
	ContainerID *cproto.ID
}

// agentID is the agent id type.
type agentID string

// agentSnapshot is a database representation of `agentState`.
type agentSnapshot struct {
	bun.BaseModel `bun:"table:resourcemanagers_agent_agentstate,alias:rmas"`

	ID                    int64       `bun:"id,pk,autoincrement"`
	AgentID               agentID     `bun:"agent_id,notnull,unique"`
	UUID                  string      `bun:"uuid,notnull,unique"`
	ResourcePoolName      string      `bun:"resource_pool_name,notnull"`
	Label                 string      `bun:"label"`
	UserEnabled           bool        `bun:"user_enabled"`
	UserDraining          bool        `bun:"user_draining"`
	MaxZeroSlotContainers int         `bun:"max_zero_slot_containers"`
	Slots                 []slotData  `bun:"slots"`
	Containers            []cproto.ID `bun:"containers"`
}

// containerSnapshot is a database representation of `containerResources`.
type containerSnapshot struct {
	bun.BaseModel `bun:"table:resourcemanagers_agent_containers,alias:rmac"`

	ResourceID sproto.ResourcesID `bun:"resource_id"`
	AgentID    aproto.ID          `bun:"agent_id"`
	ID         cproto.ID          `bun:"container_id" json:"id"`
	State      cproto.State       `bun:"state"        json:"state"`
	Devices    []device.Device    `bun:"devices"      json:"devices"`

	// Relations
	ResourcesWithState taskmodel.ResourcesWithState `bun:"rel:belongs-to,join:resource_id=resource_id"`
}

// newContainerSnapshot creates an instance from `cproto.Container`.
func newContainerSnapshot(c *cproto.Container) containerSnapshot {
	return containerSnapshot{
		ID:      c.ID,
		State:   c.State,
		Devices: c.Devices,
	}
}

// ToContainer converts to `cproto.Container`.
func (cs *containerSnapshot) ToContainer() cproto.Container {
	return cproto.Container{
		ID:      cs.ID,
		State:   cs.State,
		Devices: cs.Devices,
	}
}
