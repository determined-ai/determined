package container

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
)

// Container tracks a container running in the cluster.
type Container struct {
	// Parent stores the task handler actor address.
	Parent  actor.Address   `json:"parent"`
	ID      ID              `json:"id"`
	State   State           `json:"state"`
	Devices []device.Device `json:"devices"`
}

// New returns a new pending container.
func New(parent actor.Address, devices []device.Device, recoverable bool) Container {
	return Container{Parent: parent, ID: NewID(), State: Assigned, Devices: devices}
}

// Transition transitions the container state to the new state. An illegal transition will panic.
func (c Container) Transition(new State) Container {
	check.Panic(c.State.checkTransition(new))
	return Container{
		Parent: c.Parent, ID: c.ID, State: new, Devices: c.Devices}
}

// GPUDeviceUUIDs returns the UUIDs of the devices for this container that are GPUs.
func (c Container) GPUDeviceUUIDs() []string {
	var uuids []string
	for _, d := range c.Devices {
		if d.Type == device.GPU {
			uuids = append(uuids, d.UUID)
		}
	}
	return uuids
}

// Proto returns the proto representation of the container.
func (c *Container) Proto() *containerv1.Container {
	if c == nil {
		return nil
	}
	var devices []*devicev1.Device
	for _, d := range c.Devices {
		devices = append(devices, d.Proto())
	}
	return &containerv1.Container{
		Parent:  c.Parent.String(),
		Id:      c.ID.String(),
		State:   c.State.Proto(),
		Devices: devices,
	}
}
