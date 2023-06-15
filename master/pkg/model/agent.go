package model

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
)

// AgentSummary summarizes the state on an agent.
type AgentSummary struct {
	ID             string       `json:"id"`
	RegisteredTime time.Time    `json:"registered_time"`
	Slots          SlotsSummary `json:"slots"`
	NumContainers  int          `json:"num_containers"`
	ResourcePool   []string     `json:"resource_pool"`
	Addresses      []string     `json:"addresses"`
	Enabled        bool         `json:"enabled"`
	Draining       bool         `json:"draining"`
	Version        string       `json:"version"`
}

// ToProto converts an agent summary to a proto struct.
func (a AgentSummary) ToProto() *agentv1.Agent {
	slots := make(map[string]*agentv1.Slot)
	containers := make(map[string]*containerv1.Container)
	for i, s := range a.Slots {
		sp := s.ToProto()
		slots[i] = sp
		if sp.Container != nil {
			containers[sp.Container.Id] = sp.Container
		}
	}

	return &agentv1.Agent{
		Id:             a.ID,
		RegisteredTime: protoutils.ToTimestamp(a.RegisteredTime),
		Slots:          slots,
		Containers:     containers,
		ResourcePools:  a.ResourcePool,
		Addresses:      a.Addresses,
		Enabled:        a.Enabled,
		Draining:       a.Draining,
		Version:        a.Version,
	}
}

// AgentsSummary is a map of agent IDs to a summary of the agent.
type AgentsSummary map[string]AgentSummary

// SlotsSummary contains a summary for a number of slots.
type SlotsSummary map[string]SlotSummary

// SlotSummary summarizes the state of a slot.
type SlotSummary struct {
	ID        string            `json:"id"`
	Device    device.Device     `json:"device"`
	Enabled   bool              `json:"enabled"`
	Container *cproto.Container `json:"container"`
	Draining  bool              `json:"draining"`
}

// ToProto converts a SlotSummary to its protobuf representation.
func (s SlotSummary) ToProto() *agentv1.Slot {
	return &agentv1.Slot{
		Id:        s.ID,
		Device:    s.Device.Proto(),
		Enabled:   s.Enabled,
		Container: s.Container.ToProto(),
		Draining:  s.Draining,
	}
}

// AgentStats stores the start/end status of instance.
type AgentStats struct {
	ResourcePool string `db:"resource_pool"`
	AgentID      string `db:"agent_id"`
	Slots        int    `db:"slots"`
}
