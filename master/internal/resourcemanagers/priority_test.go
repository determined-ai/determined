package resourcemanagers

import (
	"fmt"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestGetAllPendingZeroSlotTasks(t *testing.T) {
	agents := make([]*mockAgent, 0)
	groups := []*mockGroup{
		{id: "group1", maxSlots: newMaxSlot(1), weight: 1},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0]},
		{id: "task3", slotsNeeded: 0, group: groups[0]},
		{id: "task4", slotsNeeded: 0, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, _, _ := setupSchedulerStates(t, system, tasks, groups, agents)

	allocationRequests := make(map[string]*AllocateRequest)
	for it := taskList.iterator(); it.next(); {
		req := it.value()
		allocationRequests[req.Name] = req
	}

	pendingZeroSlotTasks := getAllPendingZeroSlotTasks(taskList, "")
	expectedZeroSlotTasksIds := []*mockTask{tasks[2], tasks[3]}
	assertEqualToAllocate(t, pendingZeroSlotTasks, expectedZeroSlotTasksIds)

	task4, _ := taskList.GetTaskByID("task4")
	taskList.SetAllocations(
		task4.TaskActor,
		&ResourcesAllocated{
			task4.ID,
			"",
			[]Allocation{containerAllocation{}},
		},
	)

	pendingZeroSlotTasks = getAllPendingZeroSlotTasks(taskList, "")
	expectedZeroSlotTasksIds = []*mockTask{tasks[2]}
	assertEqualToAllocate(t, pendingZeroSlotTasks, expectedZeroSlotTasksIds)
}

func TestSortTasksByPriorityAndTimestamps(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := make([]*mockAgent, 0)
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0]},
		{id: "task3", slotsNeeded: 0, group: groups[1]},
		{id: "task4", slotsNeeded: 0, group: groups[1]},
		{id: "task5", slotsNeeded: 4, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, mockGroups, _ := setupSchedulerStates(t, system, tasks, groups, agents)

	pendingTasksByPriority, _ := sortTasksByPriorityAndTimestamp(taskList, mockGroups, "")
	for priority, tasksInPriority := range pendingTasksByPriority {
		switch priority {
		case lowerPriority:
			expectedTasks := []*mockTask{tasks[0], tasks[1]}
			assertEqualToAllocate(t, tasksInPriority, expectedTasks)
		case higherPriority:
			expectedTasks := []*mockTask{tasks[4]}
			assertEqualToAllocate(t, tasksInPriority, expectedTasks)
		default:
			panic("unexpected priority")
		}
	}

	task5, _ := taskList.GetTaskByID("task5")
	taskList.SetAllocations(
		task5.TaskActor,
		&ResourcesAllocated{
			task5.ID,
			"",
			[]Allocation{containerAllocation{}},
		},
	)

	_, scheduledTasksByPriority := sortTasksByPriorityAndTimestamp(taskList, mockGroups, "")
	for priority, tasksInPriority := range scheduledTasksByPriority {
		switch priority {
		case lowerPriority:
			expectedTasks := make([]*mockTask, 0)
			assertEqualToAllocate(t, tasksInPriority, expectedTasks)
		case higherPriority:
			expectedTasks := []*mockTask{tasks[4]}
			assertEqualToAllocate(t, tasksInPriority, expectedTasks)
		default:
			panic("unexpected priority")
		}
	}
}

func TestPrioritySchedulingNoPreemption(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
		{id: "agent2", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0]},
		{id: "task3", slotsNeeded: 1, group: groups[1]},
		{id: "task4", slotsNeeded: 0, group: groups[1]},
		{id: "task5", slotsNeeded: 4, group: groups[1]},
		{id: "task6", slotsNeeded: 0, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	// Check that agent stat has not changed.
	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}
}

func TestPrioritySchedulingNoPreemptionCase2(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
		{id: "agent2", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0]},
		{id: "task3", slotsNeeded: 12, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	// Check that agent stat has not changed.
	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}
}

func TestPrioritySchedulingNoPreemptionWithLabels(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, label: "label1"},
		{id: "agent2", slots: 4, label: "label1"},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0], label: "label1"},
		{id: "task2", slotsNeeded: 1, group: groups[0], label: "label1"},
		{id: "task3", slotsNeeded: 4, group: groups[1], label: "label2"},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[0], tasks[1]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemption(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0],
			allocatedAgent: agents[0], containerStarted: true},
		{id: "task2", slotsNeeded: 0, group: groups[0]},
		{id: "task3", slotsNeeded: 4, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	for _, agent := range agentMap {
		fmt.Println("agent has free slots: ", agent.numEmptySlots())
	}

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	expectedToRelease := []*mockTask{tasks[0]}
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestPrioritySchedulingNoSchedule(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
		{id: "task2", slotsNeeded: 8, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	for _, agent := range agentMap {
		fmt.Println("agent has free slots: ", agent.numEmptySlots())
	}

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := make([]*mockTask, 0)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}
