package agentrm

import (
	"fmt"
	"sort"
	"time"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/model"
)

type fairShare struct {
	allocationTimeout time.Duration
	scheduledReqTime  map[*sproto.AllocateRequest]time.Time
}

// NewFairShareScheduler creates a new scheduler that schedules tasks according to the max-min
// fairness of groups. For groups that are above their fair share, the scheduler requests
// them to terminate their idle tasks until they have achieved their fair share.
func NewFairShareScheduler(config *config.SchedulerConfig) Scheduler {
	var allocationTimeout time.Duration
	if config.AllocationTimeout != nil {
		allocationTimeout = time.Duration(*config.AllocationTimeout)
	}
	return &fairShare{
		allocationTimeout: allocationTimeout,
		scheduledReqTime:  make(map[*sproto.AllocateRequest]time.Time),
	}
}

type groupState struct {
	*tasklist.Group

	disabled bool
	// slotDemand is the number of slots that the group needs to run all tasks associated with
	// this group.
	slotDemand int
	// activeSlots is the number of slots in use by running tasks that can potentially be freed.
	activeSlots int
	// presubscribedSlots are slots that are already allocated and cannot be terminated.
	presubscribedSlots int
	// offered is the number of slots that were offered to the group for scheduling.
	offered int

	// reqs contains the contents of both pendingReqs and allocatedReqs.
	reqs          []*sproto.AllocateRequest
	pendingReqs   []*sproto.AllocateRequest
	allocatedReqs []*sproto.AllocateRequest
}

func (g groupState) String() string {
	address := ""
	if g.Handler != nil {
		address = fmt.Sprint("", g.Handler.Address())
	}
	return fmt.Sprintf("Group %s: disabled %v, slotDemand %v, activeSlots %v, offered %v",
		address, g.disabled, g.slotDemand, g.activeSlots, g.offered)
}

func (f *fairShare) Schedule(ctx *actor.Context, rp *resourcePool) ([]*sproto.AllocateRequest, []*actor.Ref) {
	return f.fairshareSchedule(
		ctx,
		rp.taskList,
		rp.groups,
		rp.agentStatesCache,
		rp.fittingMethod,
		rp.config.Scheduler.AllowHeterogeneousFits,
	)
}

func (f *fairShare) createJobQInfo(
	ctx *actor.Context,
	taskList *tasklist.TaskList,
) sproto.AQueue {
	reqs := make(tasklist.AllocReqs, 0, taskList.Len())
	for it := taskList.Iterator(); it.Next(); {
		reqs = append(reqs, it.Value())
	}

	jobQ := tasklist.ReduceToJobQInfo(reqs)
	for _, j := range jobQ {
		j.JobsAhead = -1 // unsupported.
	}
	return jobQ
}

func (f *fairShare) JobQInfo(ctx *actor.Context, rp *resourcePool) map[model.JobID]*sproto.RMJobInfo {
	jobQ := f.createJobQInfo(ctx, rp.taskList)
	return jobQ
}

func (f *fairShare) fairshareSchedule(
	ctx *actor.Context,
	taskList *tasklist.TaskList,
	groups map[*actor.Ref]*tasklist.Group,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
	allowHeterogeneousAgentFits bool,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	ctx.Log().Debugf("proccessing taskList of size %d", taskList.Len())
	ctx.Log().Debugf("proccessing groups of size %d", len(groups))
	ctx.Log().Debugf("proccessing agents of size %d", len(agents))

	allToAllocate := make([]*sproto.AllocateRequest, 0)
	allToRelease := make([]*actor.Ref, 0)

	for it := taskList.Iterator(); it.Next(); {
		req := it.Value()
		ctx.Log().Debugf("proccessing req %#v", req)

		allocations := taskList.Allocation(req.AllocationRef)
		if req.SlotsNeeded == 0 && allocations == nil {
			if fits := findFits(
				req,
				agents,
				fittingMethod,
				allowHeterogeneousAgentFits,
			); len(fits) == 0 {
				continue
			}
			allToAllocate = append(allToAllocate, req)
		}
	}

	ctx.Log().Debugf("current allToAllocate of size %d (after 0 slots needed)", len(allToAllocate))

	// Fair share allocations are calculated in four parts:
	// 1) Organize tasks into groups.
	// 2) Calculate the slot demand of each group.
	// 3) Allocate slot offers to each group.
	// 4) Get scheduler decisions for each group based on its slot demand.

	// TODO (sidneyw): temporarily we partition the cluster by agent label as a
	// work around for DET-1997. Tasks are given slot offers but may fail to
	// fit on any agent due to hard contraints. This may cause the scheduler to
	// not schedule any tasks and therefore not make progress. Slot offers and
	// reclaiming slots should be rethought in scheduler v2.
	capacity := totalCapacity(agents)
	groupStates := f.calculateGroupStates(ctx, taskList, groups, capacity)

	allocateSlotOffers(ctx, groupStates, capacity)
	toAllocate, toRelease := f.assignTasks(
		ctx,
		agents,
		groupStates,
		fittingMethod,
		allowHeterogeneousAgentFits,
	)
	allToAllocate = append(allToAllocate, toAllocate...)
	allToRelease = append(allToRelease, toRelease...)

	return allToAllocate, allToRelease
}

func totalCapacity(agents map[*actor.Ref]*agentState) int {
	result := 0

	for _, agent := range agents {
		result += agent.numSlots()
	}

	return result
}

func (f *fairShare) requestTimedout(req *sproto.AllocateRequest) bool {
	if f.allocationTimeout == 0 {
		return false
	}
	if req.State != sproto.SchedulingStateQueued {
		return false
	}
	startTime, ok := f.scheduledReqTime[req]
	if !ok {
		return false
	}
	return time.Since(startTime) > f.allocationTimeout
}

func (f *fairShare) calculateGroupStates(
	ctx *actor.Context,
	taskList *tasklist.TaskList,
	groups map[*actor.Ref]*tasklist.Group,
	capacity int,
) []*groupState {
	// Group all tasks by their respective task group and calculate the slot demand of each group.
	// Demand is calculated by summing the slots needed for each schedulable task.
	states := []*groupState{}
	groupMapping := make(map[*tasklist.Group]*groupState)
	for it := taskList.Iterator(); it.Next(); {
		req := it.Value()
		if req.SlotsNeeded == 0 || req.SlotsNeeded > capacity {
			continue
		}
		group := groups[req.Group]
		state, ok := groupMapping[group]
		if !ok {
			state = &groupState{
				Group:    group,
				disabled: false,
			}
			states = append(states, state)
			groupMapping[group] = state
		}
		state.reqs = append(state.reqs, req)
	}
	for _, state := range states {
		check.Panic(check.True(state.Group != nil, "the group of a task must not be nil"))
		// ctx.Log().Debugf("proccessing state %#v", state)
		for _, req := range state.reqs {
			allocated := taskList.Allocation(req.AllocationRef)
			ctx.Log().Debugf("proccessing %t req %s", allocated != nil, req.AllocationID)
			state.slotDemand += req.SlotsNeeded
			switch {
			case !tasklist.AssignmentIsScheduled(allocated):
				ctx.Log().Debugf("proccessing req %s as pending", req.AllocationID)
				state.pendingReqs = append(state.pendingReqs, req)
				if _, ok := f.scheduledReqTime[req]; !ok {
					f.scheduledReqTime[req] = time.Now()
				}
			default:
				ctx.Log().Debugf("proccessing req %s as allocated", req.AllocationID)
				if !req.Preemptible {
					state.presubscribedSlots += req.SlotsNeeded
				}
				state.allocatedReqs = append(state.allocatedReqs, req)
				// Though it would be nice if group state slot counts were counted precisely by
				// len(allocated.Resources.AgentDevices) after the incremental release feature,
				// we would also need to change other slot-related variables to be similarly
				// calculated and, unfortunately, all over the fair share code, slot demand,
				// active slots, scheduled slots and more is have many sources of truth that
				// make this change difficult without introducing bugs.
				state.activeSlots += req.SlotsNeeded
			}
		}
		if state.MaxSlots != nil {
			state.slotDemand = mathx.Min(state.slotDemand, *state.MaxSlots)
		}
	}

	return states
}

func getTotalWeight(states []*groupState) float64 {
	total := 0.0
	for _, state := range states {
		if !state.disabled && state.offered < state.slotDemand {
			total += state.Weight
		}
	}
	return total
}

func accountForPreoffers(preoffers int, offer int) (int, int) {
	if preoffers > 0 {
		if preoffers == offer {
			preoffers = 0
			offer = 0
		}
		if preoffers > offer {
			preoffers -= offer
			offer = 0
		}
		if preoffers < offer {
			preoffers = 0
			offer -= preoffers
		}
	}
	return preoffers, offer
}

func allocateSlotOffers(ctx *actor.Context, states []*groupState, capacity int) {
	ctx.Log().Debugf("allocating slot offers for %d states and capacity %d", len(states), capacity)
	// To prevent becoming oversubscribed, we first need to account for slots that were already
	// allocated to tasks that cannot be preempted.
	preoffers := make(map[*groupState]int)
	for _, state := range states {
		if state.presubscribedSlots == 0 {
			continue
		}
		// if state.presubscribedSlots > capacity, we are oversubscribed
		// This shouldn't happen outside of unit tests
		state.offered = state.presubscribedSlots
		preoffers[state] = state.presubscribedSlots
		capacity -= state.presubscribedSlots
		ctx.Log().Debugf("preoffering %d slots to group %#v", state.presubscribedSlots, state.Group)
	}

	// Slots are offered to each group based on the progressive filling algorithm, an
	// implementation of max-min fairness. All groups start with no slots offered. All
	// groups offers increase equally until groups have reached their slot demand. The
	// remaining groups have their offers increased equally. This is repeated until all slots
	// have been offered or all slot demands have been reached.

	// Due to the indivisible nature of slots, offers may not be exactly equal. Additionally, because
	// progressive filling requires groups be sorted by increasing slot demand, groups that
	// are have lower slot demand are biased towards during unequal offers. For example, if two
	// groups, each having 1 and 2 slots demands respectively, are fair shared across only 1
	// slot, then the group with only 1 slot demand will receive the slot offer.
	sort.Slice(states, func(i, j int) bool {
		first, second := states[i], states[j]
		if first.slotDemand != second.slotDemand {
			return first.slotDemand < second.slotDemand
		}
		return first.Handler.RegisteredTime().Before(second.Handler.RegisteredTime())
	})

	byTime := make([]*groupState, len(states))
	copy(byTime, states)
	sort.Slice(byTime, func(i, j int) bool {
		return states[i].Handler.RegisteredTime().After(states[j].Handler.RegisteredTime())
	})

	// To avoid any precision issues that could arise from weights of widely differing magnitudes, we
	// will recompute the total weight each time a task is removed from consideration, rather than
	// subtracting from the total.
	totalWeight := getTotalWeight(states)

	for statesLeft := len(states); statesLeft > 0; {
		// We ensure a minimum of 1 slot is fair shared in order to move progressive filling forward.
		progressMade := false
		startCapacity := capacity
		for _, state := range states {
			if state.disabled || state.offered == state.slotDemand {
				continue
			}
			calculatedFairShare := mathx.Max(
				1,
				int(float64(startCapacity)*state.Weight/totalWeight),
			)
			ctx.Log().Debugf("calculated fair share for group %p: %d", state.Group, calculatedFairShare)
			ctx.Log().Debugf("capacity: %d", capacity)
			ctx.Log().Debugf("state.slotDemand: %d", state.slotDemand)
			ctx.Log().Debugf("state.offered: %d", state.offered)
			progressMade = true
			offer := mathx.Min(calculatedFairShare, capacity, state.slotDemand-state.offered)
			ctx.Log().Debugf("offering %d slots to group %p", offer, state.Group)
			preoffers[state], offer = accountForPreoffers(preoffers[state], offer)
			ctx.Log().Debugf("offering %d slots to group %p after accounting for preoffers", offer, state.Group)
			state.offered += offer
			capacity -= offer
			if state.offered == state.slotDemand {
				statesLeft--
				totalWeight = getTotalWeight(states)
			}
		}

		if capacity == 0 {
			// We potentially need to remove deadlock between multi-slot tasks. There may be a scenario
			// where all groups will get slots below their fair share resulting in no group ever
			// launching tasks.

			// Look for the last (lowest-priority) group that does not have enough offered slots to
			// schedule any tasks at all. If there is such a group, disable it and make its offered slots
			// available for sharing again.
			adjusted := false
			for _, state := range byTime {
				smallestAllocatableTask := calculateSmallestAllocatableTask(state)
				if !state.disabled && state.offered != state.slotDemand &&
					smallestAllocatableTask != nil &&
					smallestAllocatableTask.SlotsNeeded > state.offered {
					capacity += state.offered
					state.offered = 0
					state.disabled = true
					adjusted = true
					statesLeft--
					totalWeight = getTotalWeight(states)
					break
				}
			}
			if !adjusted {
				return
			}
		} else if !progressMade {
			return
		}
	}
}

func calculateSmallestAllocatableTask(state *groupState) (smallest *sproto.AllocateRequest) {
	for _, req := range state.pendingReqs {
		if smallest == nil || req.SlotsNeeded < smallest.SlotsNeeded {
			smallest = req
		}
	}
	return smallest
}

func (f *fairShare) assignTasks(
	ctx *actor.Context,
	agents map[*actor.Ref]*agentState,
	states []*groupState,
	fittingMethod SoftConstraint,
	allowHetergenousAgentFits bool,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	toAllocate := make([]*sproto.AllocateRequest, 0)
	toRelease := make([]*actor.Ref, 0)

	for _, state := range states {
		ctx.Log().Debugf("assignTasks evaluating state: %#v", state)

		if state.activeSlots > state.offered {
			// Terminate tasks while the count of slots consumed by active tasks is greater than
			// the count of offered slots.
			// TODO: We should terminate running tasks more intelligently.
			for _, req := range state.allocatedReqs {
				if f.requestTimedout(req) || req.Preemptible {
					toRelease = append(toRelease, req.AllocationRef)
					delete(f.scheduledReqTime, req)
					state.activeSlots -= req.SlotsNeeded
					if state.activeSlots <= state.offered {
						break
					}
				}
			}
		} else if state.activeSlots < state.offered {
			// Start tasks while there are still offered slots remaining. Because slots are not
			// freed immediately, we cannot terminate and start tasks in the same scheduling call.
			state.offered -= state.activeSlots
			for _, req := range state.pendingReqs {
				if req.SlotsNeeded <= state.offered {
					if fits := findFits(
						req,
						agents,
						fittingMethod,
						allowHetergenousAgentFits,
					); len(fits) == 0 {
						continue
					}
					toAllocate = append(toAllocate, req)
					state.offered -= req.SlotsNeeded
				}
			}
		}
	}
	return toAllocate, toRelease
}
