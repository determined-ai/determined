package resourcemanagers

import (
	"fmt"
	"sort"

	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
)

type fairShareScheduler struct{}

// NewFairShareScheduler creates a new scheduler that schedules tasks according to the max-min
// fairness of groups. For groups that are above their fair share, the scheduler requests
// them to terminate their idle tasks until they have achieved their fair share.
func NewFairShareScheduler() Scheduler {
	return &fairShareScheduler{}
}

type groupState struct {
	*group

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
	if g.handler != nil {
		address = fmt.Sprint("", g.handler.Address())
	}
	return fmt.Sprintf("Group %s: disabled %v, slotDemand %v, activeSlots %v, offered %v",
		address, g.disabled, g.slotDemand, g.activeSlots, g.offered)
}

func (f *fairShareScheduler) Schedule(rp *ResourcePool) ([]*sproto.AllocateRequest, []*actor.Ref) {
	defer f.updateJobs(rp)
	return f.fairshareSchedule(rp.taskList, rp.groups, rp.agents, rp.fittingMethod)
}

func (f *fairShareScheduler) createJobQInfo(
	taskList *taskList,
) (job.AQueue, map[model.JobID]*actor.Ref) {
	reqs := make(AllocReqs, 0)
	for _, req := range taskList.taskByID {
		reqs = append(reqs, req)
	}
	jobQ, jobActors := reduceToJobQInfo(reqs)
	for _, j := range jobQ {
		j.JobsAhead = -1 // unsupported.
	}
	return jobQ, jobActors
}

func (f *fairShareScheduler) JobQInfo(rp *ResourcePool) map[model.JobID]*job.RMJobInfo {
	jobQ, _ := f.createJobQInfo(rp.taskList)
	return jobQ
}

func (f *fairShareScheduler) updateJobs(rp *ResourcePool) {
	jobQ, jobActors := f.createJobQInfo(rp.taskList)
	for jobID, jobActor := range jobActors {
		jobActor.System().Tell(jobActor, jobQ[jobID])
	}
}

func fairshareSchedule(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	f := fairShareScheduler{}
	return f.fairshareSchedule(taskList, groups, agents, fittingMethod)
}

func getTotalWeight(states []*groupState) float64 {
	total := 0.0
	for _, state := range states {
		if !state.disabled && state.offered < state.slotDemand {
			total += state.weight
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

func allocateSlotOffers(states []*groupState, capacity int) {
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
		return first.handler.RegisteredTime().Before(second.handler.RegisteredTime())
	})

	byTime := make([]*groupState, len(states))
	copy(byTime, states)
	sort.Slice(byTime, func(i, j int) bool {
		return states[i].handler.RegisteredTime().After(states[j].handler.RegisteredTime())
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
			calculatedFairShare := max(1, int(float64(startCapacity)*state.weight/totalWeight))

			progressMade = true
			offer := min(calculatedFairShare, capacity, state.slotDemand-state.offered)
			preoffers[state], offer = accountForPreoffers(preoffers[state], offer)
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

func assignTasks(
	agents map[*actor.Ref]*agentState, states []*groupState, fittingMethod SoftConstraint,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	toAllocate := make([]*sproto.AllocateRequest, 0)
	toRelease := make([]*actor.Ref, 0)

	for _, state := range states {
		if state.activeSlots > state.offered {
			// Terminate tasks while the count of slots consumed by active tasks is greater than
			// the count of offered slots.
			// TODO: We should terminate running tasks more intelligently.
			for _, req := range state.allocatedReqs {
				if req.Preemptible {
					toRelease = append(toRelease, req.TaskActor)
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
					fits := findFits(req, agents, fittingMethod)
					if len(fits) == 0 {
						continue
					}
					addTaskToAgents(fits)
					toAllocate = append(toAllocate, req)
					state.offered -= req.SlotsNeeded
				}
			}
		}
	}
	return toAllocate, toRelease
}

func (f *fairShareScheduler) fairshareSchedule(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	toAllocate := make([]*sproto.AllocateRequest, 0)
	toRelease := make([]*actor.Ref, 0)

	for label, agentsWithLabels := range splitAgentsByLabel(agents) {
		{
			// Handle zero slots.
			allocate, release := f.fairshareScheduleWithFilterZeroSlots(
				taskList, groups, agentsWithLabels, fittingMethod,
				taskFilter(label, true),
			)

			toAllocate = append(toAllocate, allocate...)
			toRelease = append(toRelease, release...)
		}

		{
			// Handle non-zero slots.
			allocate, release := f.fairshareScheduleWithFilter(
				taskList, groups, agentsWithLabels, fittingMethod,
				taskFilter(label, false),
			)

			toAllocate = append(toAllocate, allocate...)
			toRelease = append(toRelease, release...)
		}
	}

	return toAllocate, toRelease
}

func totalCapacity(agents map[*actor.Ref]*agentState) int {
	capacity := 0

	for _, agent := range agents {
		capacity += agent.numSlots()
	}

	return capacity
}

func calculateGroupStateWithFilter(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	capacity int,
	filter func(*sproto.AllocateRequest) bool,
) []*groupState {
	states := make([]*groupState, 0)
	groupMapping := make(map[*group]*groupState)

	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if !filter(req) {
			continue
		}

		if req.SlotsNeeded > capacity {
			continue
		}
		group := groups[req.Group]
		state, ok := groupMapping[group]
		if !ok {
			state = &groupState{
				group:    group,
				disabled: false,
			}
			states = append(states, state)
			groupMapping[group] = state
		}
		state.reqs = append(state.reqs, req)
	}

	for _, state := range states {
		check.Panic(check.True(state.group != nil, "the group of a task must not be nil"))
		for _, req := range state.reqs {
			allocated := taskList.GetAllocations(req.TaskActor)
			state.slotDemand += req.SlotsNeeded
			switch {
			case allocated == nil || len(allocated.Reservations) == 0:
				state.pendingReqs = append(state.pendingReqs, req)
			case len(allocated.Reservations) > 0:
				if !req.Preemptible {
					state.presubscribedSlots += req.SlotsNeeded
				}
				state.allocatedReqs = append(state.allocatedReqs, req)
				state.activeSlots += req.SlotsNeeded
			}
		}
		if state.maxSlots != nil {
			state.slotDemand = min(state.slotDemand, *state.maxSlots)
		}
	}

	return states
}

func (f *fairShareScheduler) fairshareScheduleWithFilter(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
	filter func(*sproto.AllocateRequest) bool,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	localAgentsState := deepCopyAgents(agents)

	capacity := totalCapacity(localAgentsState)
	states := calculateGroupStateWithFilter(taskList, groups, capacity, filter)
	allocateSlotOffers(states, capacity)
	toAllocate, toRelease := assignTasks(localAgentsState, states, fittingMethod)

	return toAllocate, toRelease
}

func (f *fairShareScheduler) fairshareScheduleWithFilterZeroSlots(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
	filter func(*sproto.AllocateRequest) bool,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	toAllocate := make([]*sproto.AllocateRequest, 0)
	localAgentsState := deepCopyAgents(agents)

	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if !filter(req) {
			continue
		}

		allocations := taskList.GetAllocations(req.TaskActor)
		if req.SlotsNeeded == 0 && allocations == nil {
			fits := findFits(req, localAgentsState, fittingMethod)
			if len(fits) == 0 {
				continue
			}
			addTaskToAgents(fits)
			toAllocate = append(toAllocate, req)
		}
	}

	return toAllocate, []*actor.Ref{}
}
