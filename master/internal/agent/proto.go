package agent

import (
	"github.com/determined-ai/determined/master/pkg/protoutils"
	proto "github.com/determined-ai/determined/proto/pkg/agentv1"
)

// ToProtoAgent converts an agent summary to a proto struct.
func ToProtoAgent(a AgentSummary) *proto.Agent {
	slots := make(map[string]*proto.Slot)
	for _, s := range a.Slots {
		slots[s.ID] = toProtoSlot(s)
	}
	return &proto.Agent{
		Id:             a.ID,
		RegisteredTime: protoutils.ToTimestamp(a.RegisteredTime),
		Slots:          slots,
		Containers:     nil,
		Label:          a.Label,
		ResourcePool:   a.ResourcePool,
	}
}

func toProtoSlot(s SlotSummary) *proto.Slot {
	return &proto.Slot{
		Id:        s.ID,
		Device:    s.Device.Proto(),
		Enabled:   s.Enabled,
		Container: s.Container.Proto(),
	}
}
