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

type slotStats map[string]*agentv1.DeviceStats

// SummarizeSlots a set of slots.
func SummarizeSlots(slots map[string]*agentv1.Slot) *agentv1.SlotStats {
	stats := agentv1.SlotStats{
		TypeStats:  make(slotStats),
		BrandStats: make(slotStats),
	}

	if len(slots) == 0 {
		return &stats
	}
	for _, slot := range slots {
		deviceType := slot.Device.Type.String()
		deviceTypeStats, ok := stats.TypeStats[deviceType]
		if !ok {
			deviceTypeStats = &agentv1.DeviceStats{
				States: make(map[string]int32),
			}
			stats.TypeStats[deviceType] = deviceTypeStats
		}
		deviceBrand := slot.Device.Brand
		deviceBrandStats, ok := stats.BrandStats[deviceBrand]
		if !ok {
			deviceBrandStats = &agentv1.DeviceStats{
				States: make(map[string]int32),
			}
			stats.BrandStats[deviceBrand] = deviceBrandStats
		}
		deviceBrandStats.Total++
		deviceTypeStats.Total++

		if !slot.Enabled {
			deviceBrandStats.Disabled++
			deviceTypeStats.Disabled++
		}
		if slot.Draining {
			deviceBrandStats.Draining++
			deviceTypeStats.Draining++
		}
		if slot.Container != nil {
			deviceBrandStats.States[slot.Container.State.String()]++
			deviceTypeStats.States[slot.Container.State.String()]++
		}
	}
	return &stats
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
		SlotStats:      SummarizeSlots(slots),
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
