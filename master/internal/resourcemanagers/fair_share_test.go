package resourcemanagers

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestFairShareMaxSlots(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent", slots: 4, label: ""},
	}
	groups := []*mockGroup{
		{id: "group1", maxSlots: newMaxSlot(1), weight: 1},
		{id: "group2"},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 1, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0]},
		{id: "task3", slotsNeeded: 1, group: groups[0]},
		{id: "task4", slotsNeeded: 1, group: groups[0]},
		{id: "task5", slotsNeeded: 1, group: groups[1]},
		{id: "task6", slotsNeeded: 1, group: groups[1]},
		{id: "task7", slotsNeeded: 1, group: groups[1]},
		{id: "task8", slotsNeeded: 1, group: groups[1]},
	}

	expectedToAllocate := []*mockTask{tasks[0], tasks[4], tasks[5], tasks[6]}
	expectedToRelease := []*mockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareWeights(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent", slots: 8, label: ""},
	}
	groups := []*mockGroup{
		{id: "group1", maxSlots: newMaxSlot(100), weight: 10},
		{id: "group2", maxSlots: newMaxSlot(100), weight: 30},
	}
	tasks := []*mockTask{
		{id: "task1", group: groups[0], slotsNeeded: 1},
		{id: "task2", group: groups[0], slotsNeeded: 1},
		{id: "task3", group: groups[0], slotsNeeded: 1},

		{id: "task4", group: groups[1], slotsNeeded: 1},
		{id: "task5", group: groups[1], slotsNeeded: 1},
		{id: "task6", group: groups[1], slotsNeeded: 1},
		{id: "task7", group: groups[1], slotsNeeded: 1},
		{id: "task8", group: groups[1], slotsNeeded: 1},
		{id: "task9", group: groups[1], slotsNeeded: 1},
		{id: "task10", group: groups[1], slotsNeeded: 1},
	}

	expectedToAllocate := []*mockTask{
		tasks[0], tasks[1], tasks[3], tasks[4], tasks[5], tasks[6], tasks[7], tasks[8],
	}
	expectedToRelease := []*mockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareMultiSlot(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent1", slots: 4},
		{id: "agent2", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1"},
		{id: "group2"},
	}
	tasks := []*mockTask{
		{id: "task1", group: groups[0], slotsNeeded: 4},
		{id: "task2", group: groups[1], slotsNeeded: 4},
	}

	expectedToAllocate := []*mockTask{tasks[0], tasks[1]}
	expectedToRelease := []*mockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareMaxSlotsReleaseAllocatedTasks(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent", slots: 4, label: ""},
	}
	groups := []*mockGroup{
		{id: "group1", maxSlots: newMaxSlot(2), weight: 1},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0]},
		{id: "task3", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0]},
		{id: "task4", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0]},
	}

	expectedToAllocate := []*mockTask{}
	expectedToRelease := []*mockTask{tasks[0], tasks[1]}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareUnscheduled(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent1", slots: 2, label: ""},
		{id: "agent2", slots: 2, label: ""},
	}
	groups := []*mockGroup{
		{id: "group1", maxSlots: newMaxSlot(2), weight: 1},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 2, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0]},
		{id: "task3", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[1]},
	}

	expectedToAllocate := []*mockTask{}
	expectedToRelease := []*mockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareMultiSlotDeadlock(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent", slots: 2, label: ""},
	}
	groups := []*mockGroup{
		{id: "group1"},
		{id: "group2"},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 2, group: groups[0]},
		{id: "task2", slotsNeeded: 2, group: groups[1]},
	}

	expectedToAllocate := []*mockTask{tasks[0]}
	expectedToRelease := []*mockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

// Test that a single task with slot demand higher than the cluster capacity does not prevent other
// tasks from being scheduled.
func TestFairShareBigTask(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent", slots: 4, label: ""},
	}
	groups := []*mockGroup{
		{id: "group1"},
		{id: "group2"},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 5, group: groups[0]},
		{id: "task2", slotsNeeded: 4, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	expectedToAllocate := []*mockTask{tasks[1]}
	expectedToRelease := []*mockTask{}

	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareActiveTasks(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent1", slots: 4, label: ""},
		{id: "agent2", slots: 3, label: ""},
	}
	groups := []*mockGroup{
		{id: "group1"},
		{id: "group2"},
		{id: "group3"},
		{id: "group4"},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 3, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[1]},
		{id: "task3", slotsNeeded: 1, group: groups[1], allocatedAgent: agents[1]},
		{id: "task4", slotsNeeded: 4, group: groups[2]},
		{id: "task5", slotsNeeded: 1, group: groups[3]},
	}

	expectedToAllocate := []*mockTask{tasks[0], tasks[1], tasks[4]}
	expectedToRelease := []*mockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareNilgroup(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent", slots: 4, label: ""},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, allocatedAgent: agents[0]},
		{id: "task2", slotsNeeded: 1, allocatedAgent: agents[0]},
	}

	expectedToAllocate := []*mockTask{}
	expectedToRelease := []*mockTask{tasks[0]}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, nil, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareLabels(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent1", slots: 4, label: "", maxZeroSlotContainers: 100},
		{id: "agent2", slots: 4, label: "label1", maxZeroSlotContainers: 100},
		{id: "agent3", slots: 4, label: "label2", maxZeroSlotContainers: 100},
	}
	groups := []*mockGroup{
		{id: "group1", maxSlots: newMaxSlot(1), weight: 1},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0], label: "label1"},
		{id: "task2", slotsNeeded: 1, group: groups[0], label: "label2"},
		{id: "task3", slotsNeeded: 0, group: groups[0], label: "label2"},
	}

	expectedToAllocate := []*mockTask{tasks[1], tasks[2]}
	expectedToRelease := []*mockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairSharePreemptible(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent", slots: 1, label: ""},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 1, allocatedAgent: agents[0]},
		{id: "task2", slotsNeeded: 1, allocatedAgent: agents[0]},
	}

	expectedToAllocate := []*mockTask{}
	expectedToRelease := []*mockTask{tasks[1]}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, nil, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareHonorsNonPreemptibleInAGroup(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent", slots: 1, label: ""},
	}
	groups := []*mockGroup{
		{id: "group1", maxSlots: newMaxSlot(2), weight: 1},
	}
	expectedToAllocate := []*mockTask{}

	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0], nonPreemptible: true},
	}
	expectedToRelease := []*mockTask{tasks[0]}
	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)

	// Repeat test in reverse order, because subtle bugs can be order-dependent
	tasks = []*mockTask{
		{id: "task1", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0], nonPreemptible: true},
		{id: "task2", slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0]},
	}
	expectedToRelease = []*mockTask{tasks[1]}
	system = actor.NewSystem(t.Name())
	taskList, groupMap, agentMap = setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease = fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareHonorsNonPreemptibleNilGroup(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent", slots: 1, label: ""},
	}
	expectedToAllocate := []*mockTask{}

	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 1, allocatedAgent: agents[0]},
		{id: "task2", slotsNeeded: 1, allocatedAgent: agents[0], nonPreemptible: true},
	}
	expectedToRelease := []*mockTask{tasks[0]}
	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, nil, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)

	// Repeat test in reverse order, because subtle bugs can be order-dependent
	tasks = []*mockTask{
		{id: "task1", slotsNeeded: 1, allocatedAgent: agents[0], nonPreemptible: true},
		{id: "task2", slotsNeeded: 1, allocatedAgent: agents[0]},
	}
	expectedToRelease = []*mockTask{tasks[1]}
	system = actor.NewSystem(t.Name())
	taskList, groupMap, agentMap = setupSchedulerStates(t, system, tasks, nil, agents)
	toAllocate, toRelease = fairshareSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}
