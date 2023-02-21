package task

import (
	"sort"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
	"github.com/determined-ai/determined/master/pkg/device"
)

// resourcesList tracks resourcesList with their state.
type resourcesList map[sproto.ResourcesID]*taskmodel.ResourcesWithState

func (rs resourcesList) append(ars map[sproto.ResourcesID]sproto.Resources) error {
	// Determined supports heterogeneous agent fits, so we order the resources by size,
	// since it is nicest for the chief to be on the largest node.
	var bySize []sproto.Resources
	for _, r := range ars {
		bySize = append(bySize, r)
	}
	sort.SliceStable(bySize, func(i, j int) bool {
		return bySize[i].Summary().Slots() < bySize[j].Summary().Slots()
	})

	start := len(rs)
	rank := 0
	for _, r := range bySize {
		summary := r.Summary()
		state := taskmodel.NewResourcesState(r, start+rank)
		if err := state.Persist(); err != nil {
			return err
		}
		rs[summary.ResourcesID] = &state
		rank++
	}

	return nil
}

func (rs resourcesList) first() *taskmodel.ResourcesWithState {
	for _, r := range rs {
		return r
	}
	return nil
}

func (rs resourcesList) firstDevice() *device.Device {
	for _, r := range rs {
		if r.Container != nil && len(r.Container.Devices) > 0 {
			return &r.Container.Devices[0]
		}
	}
	return nil
}

func (rs resourcesList) daemons() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.Daemon {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs resourcesList) started() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.Started != nil {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs resourcesList) active() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.Exited == nil {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs resourcesList) exited() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.Exited != nil {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs resourcesList) failed() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.Exited != nil && r.Exited.Failure != nil {
			nrs[id] = r
		}
	}
	return nrs
}
