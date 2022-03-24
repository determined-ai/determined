package task

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
)

type (
	// resourcesWithState is an sproto.Resources, along with its state that is tracked by the
	// allocation. The state is primarily just the updates that come in about the resources.
	resourcesWithState struct {
		sproto.Resources
		rank   int
		start  *sproto.ResourcesStarted
		exit   *sproto.ResourcesStopped
		daemon bool

		// The container state, if we're using a RM that uses containers and it was given to us.
		// This is a rip in the abstraction, remove eventually. Do not add usages.
		container *cproto.Container
	}

	// resourcesList tracks resourcesList with their state.
	resourcesList map[sproto.ResourcesID]*resourcesWithState
)

func newResourcesState(r sproto.Resources, rank int) resourcesWithState {
	return resourcesWithState{Resources: r, rank: rank}
}

func (rs resourcesList) append(ars []sproto.Resources) {
	start := len(rs)
	for rank, r := range ars {
		summary := r.Summary()
		state := newResourcesState(r, start+rank)
		rs[summary.ResourcesID] = &state
	}
}

func (rs resourcesList) first() *resourcesWithState {
	for _, r := range rs {
		return r
	}
	return nil
}

func (rs resourcesList) firstDevice() *device.Device {
	for _, r := range rs {
		if r.container != nil && len(r.container.Devices) > 0 {
			return &r.container.Devices[0]
		}
	}
	return nil
}

func (rs resourcesList) daemons() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.daemon {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs resourcesList) started() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.start != nil {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs resourcesList) exited() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.exit != nil {
			nrs[id] = r
		}
	}
	return nrs
}
