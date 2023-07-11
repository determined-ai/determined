package agentrm

import (
	"sort"

	"github.com/determined-ai/determined/master/internal/rm/tasklist"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

type roundRobinScheduler struct{}

// NewRoundRobinScheduler creates a new scheduler that schedules tasks via round-robin of groups
// sorted low to high by their current allocated slots.
func NewRoundRobinScheduler() Scheduler {
	return &roundRobinScheduler{}
}

func (p *roundRobinScheduler) Schedule(rp *resourcePool) (
	[]*sproto.AllocateRequest,
	[]model.AllocationID,
) {
	return roundRobinSchedule(
		rp.taskList,
		rp.groups,
		rp.agentStatesCache,
		rp.fittingMethod,
		rp.config.Scheduler.AllowHeterogeneousFits,
	)
}

func (p *roundRobinScheduler) JobQInfo(rp *resourcePool) map[model.JobID]*sproto.RMJobInfo {
	// not supported
	return make(map[model.JobID]*sproto.RMJobInfo)
}

func roundRobinSchedule(
	taskList *tasklist.TaskList,
	groups map[*actor.Ref]*tasklist.Group,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
	allowHeterogeneousFits bool,
) ([]*sproto.AllocateRequest, []model.AllocationID) {
	var states []*groupState
	groupMapping := make(map[*tasklist.Group]*groupState)
	for it := taskList.Iterator(); it.Next(); {
		req := it.Value()
		group := groups[req.Group]
		state, ok := groupMapping[group]
		if !ok {
			state = &groupState{Group: group}
			states = append(states, state)
			groupMapping[group] = state
		}
		switch {
		case !taskList.IsScheduled(req.AllocationID):
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
		return first.Handler.RegisteredTime().Before(second.Handler.RegisteredTime())
	})

	toAllocate := make([]*sproto.AllocateRequest, 0)
	for len(states) > 0 {
		filtered := states[:0]
		for _, state := range states {
			if len(state.pendingReqs) > 0 {
				req := state.pendingReqs[0]
				if fits := findFits(
					req,
					agents,
					fittingMethod,
					allowHeterogeneousFits,
				); len(fits) == 0 {
					continue
				}
				toAllocate = append(toAllocate, req)
				state.pendingReqs = state.pendingReqs[1:]
				filtered = append(filtered, state)
			}
		}
		states = filtered
	}

	return toAllocate, make([]model.AllocationID, 0)
}
