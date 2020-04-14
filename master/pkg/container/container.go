package container

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/device"
)

// Container tracks a container running in the cluster.
type Container struct {
	Parent      actor.Address   `json:"parent"`
	ID          ID              `json:"id"`
	State       State           `json:"state"`
	Devices     []device.Device `json:"devices"`
	Recoverable bool            `json:"recoverable"`
}

// New returns a new pending container.
func New(parent actor.Address, devices []device.Device, recoverable bool) Container {
	return Container{
		Parent: parent, ID: newID(), State: Assigned, Devices: devices, Recoverable: recoverable,
	}
}

// Transition transitions the container state to the new state. An illegal transition will panic.
func (c Container) Transition(new State) Container {
	check.Panic(c.State.checkTransition(new))
	return Container{
		Parent: c.Parent, ID: c.ID, State: new, Devices: c.Devices, Recoverable: c.Recoverable}
}
