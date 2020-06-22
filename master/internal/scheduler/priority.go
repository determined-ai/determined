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
		task := it.value()
		state, ok := groupMapping[task.group]
		if !ok {
			state = &groupState{group: task.group}
			states = append(states, state)
			groupMapping[task.group] = state
		}
		switch task.state {
		case taskPending:
			state.pendingTasks = append(state.pendingTasks, task)
		case taskRunning, taskTerminating:
			state.activeSlots += task.SlotsNeeded()
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
			if len(state.pendingTasks) > 0 {
				if ok := rp.assignTask(state.pendingTasks[0]); ok {
					state.pendingTasks = state.pendingTasks[1:]
					filtered = append(filtered, state)
				}
			}
		}
		states = filtered
	}
}
