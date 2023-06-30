package tproto

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TODO(!!!): Attempt to move all of `tproto` package back into `task`.

const (
	// KillAllocation is the signal to kill an allocation; analogous to in SIGKILL.
	KillAllocation AllocationSignal = "kill"
	// TerminateAllocation is the signal to kill an allocation; analogous to in SIGTERM.
	TerminateAllocation AllocationSignal = "terminate"
)

// AllocationReady marks an allocation as ready.
type AllocationReady struct{}

// AllocationWaiting marks an allocation as waiting.
type AllocationWaiting struct{}

// MarkResourcesDaemon marks the given reservation as a daemon. In the event of a normal exit,
// the allocation will not wait for it to exit on its own and instead will kill it and instead
// await it's hopefully quick termination.
type MarkResourcesDaemon struct {
	AllocationID model.AllocationID
	ResourcesID  sproto.ResourcesID
}

// AllocationSignal is an interface for signals that can be sent to an allocation.
type AllocationSignal string

// AllocationState requests allocation state. A copy is filled and returned.
type AllocationState struct {
	State     model.AllocationState
	Resources map[sproto.ResourcesID]sproto.ResourcesSummary
	Ready     bool

	Addresses  map[sproto.ResourcesID][]cproto.Address
	Containers map[sproto.ResourcesID][]cproto.Container // TODO(!!!): Why multiple containers?
}

// FirstContainer returns the first container in the allocation state.
func (a AllocationState) FirstContainer() *cproto.Container {
	for _, cs := range a.Containers {
		for _, c := range cs {
			return &c
		}
	}
	return nil
}

// FirstContainerAddresses returns the first container's addresses in the allocation state.
func (a AllocationState) FirstContainerAddresses() []cproto.Address {
	for _, ca := range a.Addresses {
		return ca
	}
	return nil
}

// SetAllocationProxyAddress manually sets the allocation proxy address.
type SetAllocationProxyAddress struct {
	ProxyAddress string
}
