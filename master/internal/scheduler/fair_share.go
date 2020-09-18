package scheduler

import (
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

	// tasks contains the contents of both pendingTasks and runningTasks, as well as all
	// terminating tasks.
	tasks        []*Task
	pendingTasks []*Task
	runningTasks []*Task
}

func (f *fairShare) Schedule(rp *DefaultRP) {
	for it := rp.taskList.iterator(); it.next(); {
		task := it.value()
		if task.SlotsNeeded() == 0 && task.state == taskPending {
			rp.assignTask(task)
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
		task := it.value()
		if task.state == taskTerminated ||
			task.SlotsNeeded() == 0 ||
			task.SlotsNeeded() > capacities[task.agentLabel] {
			continue
		}
		state, ok := groupMapping[task.group]
		if !ok {
			state = &groupState{
				group:    task.group,
				disabled: false,
			}
			states[task.agentLabel] = append(states[task.agentLabel], state)
			groupMapping[task.group] = state
		}
		state.tasks = append(state.tasks, task)
	}
	for _, group := range states {
		for _, state := range group {
			for _, task := range state.tasks {
				state.slotDemand += task.SlotsNeeded()
				switch task.state {
				case taskPending:
					state.pendingTasks = append(state.pendingTasks, task)
				case taskRunning:
					state.runningTasks = append(state.runningTasks, task)
					state.activeSlots += task.SlotsNeeded()
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
					smallestAllocatableTask.SlotsNeeded() > state.offered {
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

func calculateSmallestAllocatableTask(state *groupState) (smallest *Task) {
	for _, task := range state.pendingTasks {
		if smallest == nil || task.SlotsNeeded() < smallest.SlotsNeeded() {
			smallest = task
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
			for _, task := range state.runningTasks {
				rp.terminateTask(task, false)
				task.handler.System().Tell(task.handler, ReleaseResource{})
				if task.state == taskTerminating {
					state.activeSlots -= task.SlotsNeeded()
				}
				if state.activeSlots <= state.offered {
					break
				}
			}
		} else if state.activeSlots < state.offered {
			// Start tasks while there are still offered slots remaining. Because slots are not
			// freed immediately, we cannot terminate and start tasks in the same scheduling call.
			state.offered -= state.activeSlots
			for _, task := range state.pendingTasks {
				if task.SlotsNeeded() <= state.offered {
					if ok := rp.assignTask(task); ok {
						state.offered -= task.SlotsNeeded()
					}
				}
			}
		}
	}
}
