package agentrm

import (
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/mathx"
)

// calculateDesiredNewAgentNum calculates the new instances based on pending tasks and
// slots per instance.
func calculateDesiredNewAgentNum(
	taskList *tasklist.TaskList,
	groups map[*actor.Ref]*tasklist.Group,
	slotsPerAgent int,
	maxZeroSlotTasksPerAgent int,
) int {
	slotSum := 0
	allTasks := 0
	zeroSlotTasks := 0
	groupSlotsNeeded := make(map[*tasklist.Group]int)
	for it := taskList.Iterator(); it.Next(); {
		// TODO(DET-4035): This code is duplicated from the fitting functions in the
		//    scheduler. To determine is a task is schedulable, we would ideally interface
		//    with the scheduler in some way and not duplicate this logic.
		switch {
		case taskList.IsScheduled(it.Value().AllocationID):
			// If a task is already allocated, skip it.
			continue
		case it.Value().SlotsNeeded == 0:
			zeroSlotTasks++
			allTasks++
		case slotsPerAgent == 0:
			continue
		case it.Value().SlotsNeeded <= slotsPerAgent, it.Value().SlotsNeeded%slotsPerAgent == 0:
			if groups != nil {
				group := groups[it.Value().Group]
				groupSlotsNeeded[group] += it.Value().SlotsNeeded
			} else {
				slotSum += it.Value().SlotsNeeded
			}
			allTasks++
		}
	}

	for g, groupSlotSum := range groupSlotsNeeded {
		maxSlots := g.MaxSlots
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
