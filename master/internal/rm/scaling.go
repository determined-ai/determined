package rm

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/mathx"
)

// calculateDesiredNewAgentNum calculates the new instances based on pending tasks and
// slots per instance.
func calculateDesiredNewAgentNum(
	taskList *taskList, groups map[*actor.Ref]*group, slotsPerAgent int, maxZeroSlotTasksPerAgent int,
) int {
	slotSum := 0
	allTasks := 0
	zeroSlotTasks := 0
	groupSlotsNeeded := make(map[*group]int)
	for it := taskList.iterator(); it.next(); {
		// TODO(DET-4035): This code is duplicated from the fitting functions in the
		//    scheduler. To determine is a task is schedulable, we would ideally interface
		//    with the scheduler in some way and not duplicate this logic.
		switch {
		case taskList.GetAllocations(it.value().AllocationRef) != nil:
			// If a task is already allocated, skip it.
			continue
		case it.value().SlotsNeeded == 0:
			zeroSlotTasks++
			allTasks++
		case slotsPerAgent == 0:
			continue
		case it.value().SlotsNeeded <= slotsPerAgent, it.value().SlotsNeeded%slotsPerAgent == 0:
			if groups != nil {
				group := groups[it.value().Group]
				groupSlotsNeeded[group] += it.value().SlotsNeeded
			} else {
				slotSum += it.value().SlotsNeeded
			}
			allTasks++
		}
	}

	for g, groupSlotSum := range groupSlotsNeeded {
		maxSlots := g.maxSlots
		if maxSlots != nil {
			slotSum += mathx.Min(*maxSlots, groupSlotSum)
		} else {
			slotSum += groupSlotSum
		}
	}

	numAgentByZeroSlot, numAgentBySlot := 0, 0
	switch {
	case zeroSlotTasks == 0:
		numAgentByZeroSlot = 0
	case maxZeroSlotTasksPerAgent == 0:
		numAgentByZeroSlot = 0
	default:
		numAgentByZeroSlot = (zeroSlotTasks + maxZeroSlotTasksPerAgent - 1) / maxZeroSlotTasksPerAgent
	}
	switch {
	case slotSum == 0:
		numAgentBySlot = 0
	case slotsPerAgent == 0:
		numAgentBySlot = 0
	default:
		numAgentBySlot = (slotSum + slotsPerAgent - 1) / slotsPerAgent
	}
	return mathx.Max(numAgentByZeroSlot, numAgentBySlot)
}
