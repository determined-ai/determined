package scheduler

import (
	"fmt"
	"sort"
)

type fairShare struct{}

// NewFairShareScheduler creates a new scheduler that schedules tasks according to the max-min
// fairness of groups. For groups that are above their fair share, the scheduler requests
// them to terminate their idle tasks until they have achieved their fair share.
func NewFairShareScheduler() Scheduler {
	return &fairShare{}
}

type groupState struct {
	*group

	disabled bool
	// slotDemand is the number of slots that the group needs to run all tasks associated with
	// this group.
	slotDemand int
	// activeSlots is the number of slots in use by running tasks that can potentially be freed.
	activeSlots int
	// offered is the number of slots that were offered to the group for scheduling.
	offered int

	// reqs contains the contents of both pendingReqs and allocatedReqs.
	reqs          []*AllocateRequest
	pendingReqs   []*AllocateRequest
	allocatedReqs []*AllocateRequest
}

func (g groupState) String() string {
	address := ""
	if g.handler != nil {
		address = fmt.Sprint("", g.handler.Address())
	}
	return fmt.Sprintf("Group %s: disabled %v, slotDemand %v, activeSlots %v, offered %v",
		address, g.disabled, g.slotDemand, g.activeSlots, g.offered)
}

func (f *fairShare) Schedule(rp *DefaultRP) {
	for it := rp.taskList.iterator(); it.next(); {
		req := it.value()
		allocations := rp.taskList.GetAllocations(req.TaskActor)
		if req.SlotsNeeded == 0 && allocations == nil {
			rp.allocateResources(req)
		}
	}

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
	capacity := capacityByAgentLabel(rp)
	states := calculateGroupStates(rp, capacity)

	for label, groupStates := range states {
		allocateSlotOffers(groupStates, capacity[label])
		assignTasks(rp, groupStates)
	}
}

func capacityByAgentLabel(rp *DefaultRP) map[string]int {
	agentCap := map[string]int{}

	for _, agent := range rp.agents {
		agentCap[agent.label] += agent.numSlots()
	}

	return agentCap
}

func calculateGroupStates(rp *DefaultRP, capacities map[string]int) map[string][]*groupState {
	// Group all tasks by their respective task group and calculate the slot demand of each group.
	// Demand is calculated by summing the slots needed for each schedulable task.
	states := make(map[string][]*groupState)
	groupMapping := make(map[*group]*groupState)
	for it := rp.taskList.iterator(); it.next(); {
		req := it.value()
		if req.SlotsNeeded == 0 || req.SlotsNeeded > capacities[req.Label] {
			continue
		}
		group := rp.groups[req.Group]
		state, ok := groupMapping[group]
		if !ok {
			state = &groupState{
				group:    group,
				disabled: false,
			}
			states[req.Label] = append(states[req.Label], state)
			groupMapping[group] = state
		}
		state.reqs = append(state.reqs, req)
	}
	for _, group := range states {
		for _, state := range group {
			for _, req := range state.reqs {
				allocated := rp.taskList.GetAllocations(req.TaskActor)
				state.slotDemand += req.SlotsNeeded
				switch {
				case allocated == nil || len(allocated.Allocations) == 0:
					state.pendingReqs = append(state.pendingReqs, req)
				case len(allocated.Allocations) > 0:
					state.allocatedReqs = append(state.allocatedReqs, req)
					state.activeSlots += req.SlotsNeeded
				}
			}
			if state.maxSlots != nil {
				state.slotDemand = min(state.slotDemand, *state.maxSlots)
			}
		}
	}

	return states
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

func allocateSlotOffers(states []*groupState, capacity int) {
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

func calculateSmallestAllocatableTask(state *groupState) (smallest *AllocateRequest) {
	for _, req := range state.pendingReqs {
		if smallest == nil || req.SlotsNeeded < smallest.SlotsNeeded {
			smallest = req
		}
	}
	return smallest
}

func assignTasks(rp *DefaultRP, states []*groupState) {
	for _, state := range states {
		if state.activeSlots > state.offered {
			// Terminate tasks while the count of slots consumed by active tasks is greater than
			// the count of offered slots.
			// TODO: We should terminate running tasks more intelligently.
			for _, req := range state.allocatedReqs {
				rp.releaseResource(req.TaskActor)
				state.activeSlots -= req.SlotsNeeded
				if state.activeSlots <= state.offered {
					break
				}
			}
		} else if state.activeSlots < state.offered {
			// Start tasks while there are still offered slots remaining. Because slots are not
			// freed immediately, we cannot terminate and start tasks in the same scheduling call.
			state.offered -= state.activeSlots
			for _, req := range state.pendingReqs {
				if req.SlotsNeeded <= state.offered {
					if ok := rp.allocateResources(req); ok {
						state.offered -= req.SlotsNeeded
					}
				}
			}
		}
	}
}
