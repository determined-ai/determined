package scheduler

import (
	"sort"
)

type priorityScheduler struct{}

// NewPriorityScheduler creates a new scheduler that schedules tasks via round-robin of groups
// sorted low to high by their current allocated slots.
func NewPriorityScheduler() Scheduler {
	return &priorityScheduler{}
}

func (p *priorityScheduler) Schedule(rp *DefaultRP) {
	var states []*groupState
	groupMapping := make(map[*group]*groupState)
	for it := rp.taskList.iterator(); it.next(); {
		req := it.value()
		group := rp.groups[req.Group]
		state, ok := groupMapping[group]
		if !ok {
			state = &groupState{group: group}
			states = append(states, state)
			groupMapping[group] = state
		}
		assigned := rp.taskList.GetAllocations(req.TaskActor)
		switch {
		case assigned == nil || len(assigned.Allocations) == 0:
			state.pendingReqs = append(state.pendingReqs, req)
		default:
			state.activeSlots += req.SlotsNeeded
		}
	}

	sort.Slice(states, func(i, j int) bool {
		first, second := states[i], states[j]
		if first.activeSlots != second.activeSlots {
			return first.activeSlots < second.activeSlots
		}
		return first.handler.RegisteredTime().Before(second.handler.RegisteredTime())
	})

	for len(states) > 0 {
		filtered := states[:0]
		for _, state := range states {
			if len(state.pendingReqs) > 0 {
				if ok := rp.allocateResources(state.pendingReqs[0]); ok {
					state.pendingReqs = state.pendingReqs[1:]
					filtered = append(filtered, state)
				}
			}
		}
		states = filtered
	}
}
