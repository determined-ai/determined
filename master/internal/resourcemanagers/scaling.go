package resourcemanagers

// calculateDesiredNewInstanceNum calculates the new instances based on pending tasks and
// slots per instance.
func calculateDesiredNewInstanceNum(taskList *taskList, slotsPerInstance int) int {
	slotSum := 0
	zeroSlotTasks := false
	for it := taskList.iterator(); it.next(); {
		// TODO(DET-4035): This code is duplicated from the fitting functions in the
		//    scheduler. To determine is a task is schedulable, we would ideally interface
		//    with the scheduler in some way and not duplicate this logic.
		switch {
		case taskList.GetAllocations(it.value().TaskActor) != nil:
			// If a task is already allocated, skip it.
			continue
		case it.value().SlotsNeeded == 0:
			zeroSlotTasks = true
		case slotsPerInstance == 0:
			continue
		case it.value().SlotsNeeded <= slotsPerInstance:
			slotSum += it.value().SlotsNeeded
		case it.value().SlotsNeeded%slotsPerInstance == 0:
			slotSum += it.value().SlotsNeeded
		}
	}

	switch {
	case zeroSlotTasks && slotSum == 0:
		return 1
	case !zeroSlotTasks && slotsPerInstance == 0:
		return 0
	default:
		return (slotSum + slotsPerInstance - 1) / slotsPerInstance
	}
}
