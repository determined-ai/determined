package resourcemanagers

import (
	"sort"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

type roundRobinScheduler struct{}

// NewRoundRobinScheduler creates a new scheduler that schedules tasks via round-robin of groups
// sorted low to high by their current allocated slots.
func NewRoundRobinScheduler() Scheduler {
	return &roundRobinScheduler{}
}

func (p *roundRobinScheduler) Schedule(rp *ResourcePool) ([]*sproto.AllocateRequest, []*actor.Ref) {
	return roundRobinSchedule(rp.taskList, rp.groups, rp.agents, rp.fittingMethod)
}

func roundRobinSchedule(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	var states []*groupState
	groupMapping := make(map[*group]*groupState)
	for it := taskList.iterator(); it.next(); {
		req := it.value()
		group := groups[req.Group]
		state, ok := groupMapping[group]
		if !ok {
			state = &groupState{group: group}
			states = append(states, state)
			groupMapping[group] = state
		}
		assigned := taskList.GetAllocations(req.TaskActor)
		switch {
		case assigned == nil || len(assigned.Reservations) == 0:
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

	toAllocate := make([]*sproto.AllocateRequest, 0)
	for len(states) > 0 {
		filtered := states[:0]
		for _, state := range states {
			if len(state.pendingReqs) > 0 {
				req := state.pendingReqs[0]
				if fits := findFits(req, agents, fittingMethod); len(fits) == 0 {
					continue
				}
				toAllocate = append(toAllocate, req)
				state.pendingReqs = state.pendingReqs[1:]
				filtered = append(filtered, state)
			}
		}
		states = filtered
	}

	return toAllocate, make([]*actor.Ref, 0)
}
