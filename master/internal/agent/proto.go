package agent

import (
	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	proto "github.com/determined-ai/determined/proto/pkg/agentv1"
)

// ToProtoAgent converts agent summary to a proto struct.
func ToProtoAgent(a AgentSummary) *proto.Agent {
	slots := make(map[string]*proto.Slot)
	for _, s := range a.Slots {
		slots[s.ID] = toProtoSlot(s)
	}
	return &proto.Agent{
		Id: a.ID,
		RegisteredTime: &timestamp.Timestamp{
			Seconds: a.RegisteredTime.Unix(),
			Nanos:   int32(a.RegisteredTime.Nanosecond()),
		},
		Slots:      slots,
		Containers: nil,
		Label:      a.Label,
	}
}

func toProtoSlot(s SlotSummary) *proto.Slot {
	var c *proto.Container
	if s.Container != nil {
		c = toProtoContainer(*s.Container)
	}
	return &proto.Slot{
		Id:        s.ID,
		Device:    toProtoDevice(s.Device),
		Enabled:   s.Enabled,
		Container: c,
	}
}

func toProtoContainer(c container.Container) *proto.Container {
	var devices []*proto.Device
	for _, d := range c.Devices {
		devices = append(devices, toProtoDevice(d))
	}
	return &proto.Container{
		Parent:  c.Parent.String(),
		Id:      c.ID.String(),
		State:   toProtoContainerState(c.State),
		Devices: devices,
	}
}

func toProtoContainerState(s container.State) proto.Container_State {
	switch s {
	case container.Assigned:
		return proto.Container_STATE_ASSIGNED
	case container.Pulling:
		return proto.Container_STATE_PULLING
	case container.Starting:
		return proto.Container_STATE_STARTING
	case container.Running:
		return proto.Container_STATE_RUNNING
	case container.Terminated:
		return proto.Container_STATE_TERMINATED
	default:
		return proto.Container_STATE_UNSPECIFIED
	}
}

func toProtoDevice(d device.Device) *proto.Device {
	return &proto.Device{
		Id:    int32(d.ID),
		Brand: d.Brand,
		Uuid:  d.UUID,
		Type:  toProtoDeviceType(d.Type),
	}
}

func toProtoDeviceType(t device.Type) proto.Device_Type {
	switch t {
	case device.CPU:
		return proto.Device_TYPE_CPU
	case device.GPU:
		return proto.Device_TYPE_GPU
	default:
		return proto.Device_TYPE_UNSPECIFIED
	}
}
