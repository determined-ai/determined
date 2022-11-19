package agentrm

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
)

// SlotData is a database representation of slot state.
type SlotData struct {
	Device      device.Device
	UserEnabled bool
	ContainerID *cproto.ID
}

// AgentID is the agent id type.
type AgentID string

// AgentSnapshot is a database representation of `agentState`.
type AgentSnapshot struct {
	bun.BaseModel `bun:"table:resourcemanagers_agent_agentstate,alias:rmas"`

	ID                    int64       `bun:"id,pk,autoincrement"`
	AgentID               AgentID     `bun:"agent_id,notnull,unique"`
	UUID                  string      `bun:"uuid,notnull,unique"`
	ResourcePoolName      string      `bun:"resource_pool_name,notnull"`
	Label                 string      `bun:"label"`
	UserEnabled           bool        `bun:"user_enabled"`
	UserDraining          bool        `bun:"user_draining"`
	MaxZeroSlotContainers int         `bun:"max_zero_slot_containers"`
	Slots                 []SlotData  `bun:"slots"`
	Containers            []cproto.ID `bun:"containers"`
}

// ContainerSnapshot is a database representation of `containerResources`.
type ContainerSnapshot struct {
	bun.BaseModel `bun:"table:resourcemanagers_agent_containers,alias:rmac"`

	ResourceID sproto.ResourcesID `bun:"resource_id"`
	AgentID    aproto.ID          `bun:"agent_id"`
	ID         cproto.ID          `bun:"container_id" json:"id"`
	State      cproto.State       `bun:"state"        json:"state"`
	Devices    []device.Device    `bun:"devices"      json:"devices"`

	// Relations
	ResourcesWithState taskmodel.ResourcesWithState `bun:"rel:belongs-to,join:resource_id=resource_id"`
}

// NewContainerSnapshot creates an instance from `cproto.Container`.
func NewContainerSnapshot(c *cproto.Container) ContainerSnapshot {
	return ContainerSnapshot{
		ID:      c.ID,
		State:   c.State,
		Devices: c.Devices,
	}
}

// ToContainer converts to `cproto.Container`.
func (cs *ContainerSnapshot) ToContainer() cproto.Container {
	return cproto.Container{
		ID:      cs.ID,
		State:   cs.State,
		Devices: cs.Devices,
	}
}
