package model

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
)

// AgentSummary summarizes the state on an agent.
type AgentSummary struct {
	ID             string       `json:"id"`
	RegisteredTime time.Time    `json:"registered_time"`
	Slots          SlotsSummary `json:"slots"`
	NumContainers  int          `json:"num_containers"`
	ResourcePool   string       `json:"resource_pool"`
	Label          string       `json:"label"`
	Addresses      []string     `json:"addresses"`
	Enabled        bool         `json:"enabled"`
	Draining       bool         `json:"draining"`
}

// ToProto converts an agent summary to a proto struct.
func (a AgentSummary) ToProto() *agentv1.Agent {
	slots := make(map[string]*agentv1.Slot)
	for i, s := range a.Slots {
		slots[i] = s.ToProto()
	}
	return &agentv1.Agent{
		Id:             a.ID,
		RegisteredTime: protoutils.ToTimestamp(a.RegisteredTime),
		Slots:          slots,
		Containers:     nil,
		Label:          a.Label,
		ResourcePool:   a.ResourcePool,
		Addresses:      a.Addresses,
		Enabled:        a.Enabled,
		Draining:       a.Draining,
	}
}

// AgentsSummary is a map of agent IDs to a summary of the agent.
type AgentsSummary map[string]AgentSummary

// SlotsSummary contains a summary for a number of slots.
type SlotsSummary map[string]SlotSummary

// SlotSummary summarizes the state of a slot.
type SlotSummary struct {
	ID        string               `json:"id"`
	Device    device.Device        `json:"device"`
	Enabled   bool                 `json:"enabled"`
	Container *container.Container `json:"container"`
	Draining  bool                 `json:"draining"`
}

// ToProto converts a SlotSummary to its protobuf representation.
func (s SlotSummary) ToProto() *agentv1.Slot {
	return &agentv1.Slot{
		Id:        s.ID,
		Device:    s.Device.Proto(),
		Enabled:   s.Enabled,
		Container: s.Container.Proto(),
		Draining:  s.Draining,
	}
}
