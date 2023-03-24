package agentrm

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestFairShareMaxSlots(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1", MaxSlots: newMaxSlot(1), Weight: 1},
		{ID: "group2"},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task3", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task4", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task5", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task6", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task7", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task8", SlotsNeeded: 1, Group: groups[1]},
	}

	expectedToAllocate := []*MockTask{tasks[0], tasks[4], tasks[5], tasks[6]}
	expectedToRelease := []*MockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareWeights(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 8},
	}
	groups := []*MockGroup{
		{ID: "group1", MaxSlots: newMaxSlot(100), Weight: 10},
		{ID: "group2", MaxSlots: newMaxSlot(100), Weight: 30},
	}
	tasks := []*MockTask{
		{ID: "task1", Group: groups[0], SlotsNeeded: 1},
		{ID: "task2", Group: groups[0], SlotsNeeded: 1},
		{ID: "task3", Group: groups[0], SlotsNeeded: 1},

		{ID: "task4", Group: groups[1], SlotsNeeded: 1},
		{ID: "task5", Group: groups[1], SlotsNeeded: 1},
		{ID: "task6", Group: groups[1], SlotsNeeded: 1},
		{ID: "task7", Group: groups[1], SlotsNeeded: 1},
		{ID: "task8", Group: groups[1], SlotsNeeded: 1},
		{ID: "task9", Group: groups[1], SlotsNeeded: 1},
		{ID: "task10", Group: groups[1], SlotsNeeded: 1},
	}

	expectedToAllocate := []*MockTask{
		tasks[0], tasks[1], tasks[3], tasks[4], tasks[5], tasks[6], tasks[7], tasks[8],
	}
	expectedToRelease := []*MockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareMultiSlot(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1"},
		{ID: "group2"},
	}
	tasks := []*MockTask{
		{ID: "task1", Group: groups[0], SlotsNeeded: 4},
		{ID: "task2", Group: groups[1], SlotsNeeded: 4},
	}

	expectedToAllocate := []*MockTask{tasks[0], tasks[1]}
	expectedToRelease := []*MockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareMaxSlotsReleaseAllocatedTasks(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1", MaxSlots: newMaxSlot(2), Weight: 1},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0]},
		{ID: "task3", SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0]},
		{ID: "task4", SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0]},
	}

	expectedToAllocate := []*MockTask{}
	expectedToRelease := []*MockTask{tasks[0], tasks[1]}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareUnscheduled(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent1", Slots: 2},
		{ID: "agent2", Slots: 2},
	}
	groups := []*MockGroup{
		{ID: "group1", MaxSlots: newMaxSlot(2), Weight: 1},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 2, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0]},
		{ID: "task3", SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[1]},
	}

	expectedToAllocate := []*MockTask{}
	expectedToRelease := []*MockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareMultiSlotDeadlock(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 2},
	}
	groups := []*MockGroup{
		{ID: "group1"},
		{ID: "group2"},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 2, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 2, Group: groups[1]},
	}

	expectedToAllocate := []*MockTask{tasks[0]}
	expectedToRelease := []*MockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

// Test that a single task with slot demand higher than the cluster capacity does not prevent other
// tasks from being scheduled.
func TestFairShareBigTask(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1"},
		{ID: "group2"},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 5, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 4, Group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	expectedToAllocate := []*MockTask{tasks[1]}
	expectedToRelease := []*MockTask{}

	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareActiveTasks(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 3},
	}
	groups := []*MockGroup{
		{ID: "group1"},
		{ID: "group2"},
		{ID: "group3"},
		{ID: "group4"},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 3, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task3", SlotsNeeded: 1, Group: groups[1], AllocatedAgent: agents[1]},
		{ID: "task4", SlotsNeeded: 4, Group: groups[2]},
		{ID: "task5", SlotsNeeded: 1, Group: groups[3]},
	}

	expectedToAllocate := []*MockTask{tasks[0], tasks[1], tasks[4]}
	expectedToRelease := []*MockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareNilgroup(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 4},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, AllocatedAgent: agents[0]},
		{ID: "task2", SlotsNeeded: 1, AllocatedAgent: agents[0]},
	}

	expectedToAllocate := []*MockTask{}
	expectedToRelease := []*MockTask{tasks[0]}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, nil, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairSharePreemptible(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 1},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 1, AllocatedAgent: agents[0]},
		{ID: "task2", SlotsNeeded: 1, AllocatedAgent: agents[0]},
	}

	expectedToAllocate := []*MockTask{}
	expectedToRelease := []*MockTask{tasks[1]}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, nil, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareHonorsNonPreemptibleInAGroup(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 1},
	}
	groups := []*MockGroup{
		{ID: "group1", MaxSlots: newMaxSlot(2), Weight: 1},
	}
	expectedToAllocate := []*MockTask{}

	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0]},
		{
			ID:             "task2",
			SlotsNeeded:    1,
			Group:          groups[0],
			AllocatedAgent: agents[0],
			NonPreemptible: true,
		},
	}
	expectedToRelease := []*MockTask{tasks[0]}
	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)

	// Repeat test in reverse order, because subtle bugs can be order-dependent
	tasks = []*MockTask{
		{
			ID:             "task1",
			SlotsNeeded:    1,
			Group:          groups[0],
			AllocatedAgent: agents[0],
			NonPreemptible: true,
		},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0]},
	}
	expectedToRelease = []*MockTask{tasks[1]}
	system = actor.NewSystem(t.Name())
	taskList, groupMap, agentMap = setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease = fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareHonorsNonPreemptibleNilGroup(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 1},
	}
	expectedToAllocate := []*MockTask{}

	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 1, AllocatedAgent: agents[0]},
		{ID: "task2", SlotsNeeded: 1, AllocatedAgent: agents[0], NonPreemptible: true},
	}
	expectedToRelease := []*MockTask{tasks[0]}
	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, nil, agents)
	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)

	// Repeat test in reverse order, because subtle bugs can be order-dependent
	tasks = []*MockTask{
		{ID: "task1", SlotsNeeded: 1, AllocatedAgent: agents[0], NonPreemptible: true},
		{ID: "task2", SlotsNeeded: 1, AllocatedAgent: agents[0]},
	}
	expectedToRelease = []*MockTask{tasks[1]}
	system = actor.NewSystem(t.Name())
	taskList, groupMap, agentMap = setupSchedulerStates(t, system, tasks, nil, agents)
	toAllocate, toRelease = fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}
