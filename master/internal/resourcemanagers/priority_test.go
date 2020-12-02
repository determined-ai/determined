package resourcemanagers

import (
	"fmt"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

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

	zeroSlotPendingTasksByPriority, _ := sortTasksByPriorityAndTimestamp(
		taskList, mockGroups, "", zeroSlotTaskFilter)

	tasksInLowerPriority := zeroSlotPendingTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority := []*mockTask{}
	assertEqualToAllocate(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority := zeroSlotPendingTasksByPriority[higherPriority]
	expectedTasksInHigherPriority := []*mockTask{tasks[2], tasks[3]}
	assertEqualToAllocate(t, tasksInHigherPriority, expectedTasksInHigherPriority)

	nonZeroSlotPendingTasksByPriority, _ := sortTasksByPriorityAndTimestamp(
		taskList, mockGroups, "", nonZeroSlotTaskFilter)

	tasksInLowerPriority = nonZeroSlotPendingTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority = []*mockTask{tasks[0], tasks[1]}
	assertEqualToAllocate(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority = nonZeroSlotPendingTasksByPriority[higherPriority]
	expectedTasksInHigherPriority = []*mockTask{tasks[4]}
	assertEqualToAllocate(t, tasksInHigherPriority, expectedTasksInHigherPriority)

	forceSetTaskAllocations(t, taskList, "task5", 1)
	_, scheduledTasksByPriority := sortTasksByPriorityAndTimestamp(
		taskList, mockGroups, "", nonZeroSlotTaskFilter)

	tasksInLowerPriority = scheduledTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority = make([]*mockTask, 0)
	assertEqualToAllocate(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority = scheduledTasksByPriority[higherPriority]
	expectedTasksInHigherPriority = []*mockTask{tasks[4]}
	assertEqualToAllocate(t, tasksInHigherPriority, expectedTasksInHigherPriority)
}

func TestPrioritySchedulingPreemptionDisabled(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 100},
		{id: "agent2", slots: 4, maxZeroSlotContainers: 100},
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

func TestPrioritySchedulingPreemptionDisabledHigherPriorityBlocksLowerPriority(t *testing.T) {
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

func TestPrioritySchedulingPreemptionDisabledWithLabels(t *testing.T) {
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

func TestPrioritySchedulingPreemptionAaron(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 100},
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

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	expectedToRelease := []*mockTask{tasks[0]}
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestPrioritySchedulingLowPriorityTasksAreBlockedByHigherPriority(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 100},
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

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := make([]*mockTask, 0)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulerPreemptZeroSlotTask(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 1, zeroSlotContainers: 1},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 0, group: groups[0],
			allocatedAgent: agents[0], containerStarted: true},
		{id: "task2", slotsNeeded: 0, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	fmt.Println("1")
	expectedToAllocate := make([]*mockTask, 0)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	fmt.Println("2", toRelease)
	expectedToRelease := []*mockTask{tasks[0]}
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}


func TestPrioritySchedulingNoPreemptionAddTasks(t *testing.T) {
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

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 1, group: groups[0]},
		{id: "task8", slotsNeeded: 1, group: groups[0]},
		{id: "task9", slotsNeeded: 1, group: groups[0]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, _ = p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	expectedToAllocate = []*mockTask{newTasks[0], newTasks[1]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingNoPreemptionDepth2(t *testing.T) {
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
		{id: "task4", slotsNeeded: 1, group: groups[1]},
		{id: "task5", slotsNeeded: 4, group: groups[1]},
		{id: "task6", slotsNeeded: 1, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 1, group: groups[1]},
		{id: "task8", slotsNeeded: 1, group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, _ = p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, []*mockTask{})
}

func TestPrioritySchedulingNoPreemptionLowerPriorityMustWait(t *testing.T) {
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
		{id: "task1", slotsNeeded: 1, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[1]},
		{id: "task3", slotsNeeded: 1, group: groups[1]},
		{id: "task4", slotsNeeded: 1, group: groups[1]},
		{id: "task5", slotsNeeded: 2, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	firstAllocation, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3]}
	assertEqualToAllocate(t, firstAllocation, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(firstAllocation, agentMap, taskList)

	secondAllocation, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, secondAllocation, []*mockTask{})

	for _, task := range firstAllocation {
		RemoveTask(task.SlotsNeeded, task.TaskActor, taskList, true)
	}

	thirdAllocation, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, thirdAllocation, []*mockTask{tasks[0], tasks[4]})
}

func TestPrioritySchedulingNoPreemptionTaskFinished(t *testing.T) {
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)
	ok := RemoveTask(4, toAllocate[0].TaskActor, taskList, true)
	if !ok {
		t.Errorf("Failed to remove task %s", toAllocate[0].ID)
	}

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 1, group: groups[0]},
		{id: "task8", slotsNeeded: 1, group: groups[0]},
		{id: "task9", slotsNeeded: 0, group: groups[0]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, _ = p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, []*mockTask{newTasks[0], newTasks[1], newTasks[2]})
}

func TestPrioritySchedulingNoPreemptionAllTasksFinished(t *testing.T) {
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
		{id: "task4", slotsNeeded: 1, group: groups[1]},
		{id: "task5", slotsNeeded: 4, group: groups[1]},
		{id: "task6", slotsNeeded: 1, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 4, group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	for _, task := range toAllocate {
		RemoveTask(task.SlotsNeeded, task.TaskActor, taskList, true)
	}

	toAllocate, _ = p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, []*mockTask{tasks[0], newTasks[0]})
}

func TestPrioritySchedulingPreemptOneTask(t *testing.T) {
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
		{id: "task4", slotsNeeded: 1, group: groups[1]},
		{id: "task5", slotsNeeded: 4, group: groups[1]},
		{id: "task6", slotsNeeded: 1, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 1, group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToRelease(t, taskList, toRelease, []*mockTask{tasks[5]})
	assertEqualToAllocate(t, toAllocate, []*mockTask{})

	ok := RemoveTask(tasks[5].slotsNeeded, toRelease[0], taskList, false)
	if !ok {
		t.Errorf("Failed to remove task %s", toRelease[0].Address())
	}

	toAllocate, toRelease = p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToRelease(t, taskList, toRelease, []*mockTask{})
	assertEqualToAllocate(t, toAllocate, []*mockTask{newTasks[0]})
}

func TestPrioritySchedulingPreemptAllTasks(t *testing.T) {
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
		{id: "task3", slotsNeeded: 1, group: groups[0]},
		{id: "task4", slotsNeeded: 1, group: groups[0]},
		{id: "task5", slotsNeeded: 1, group: groups[0]},
		{id: "task6", slotsNeeded: 0, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[0], tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 4, group: groups[1]},
		{id: "task8", slotsNeeded: 4, group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	expectedToRelease := []*mockTask{tasks[0], tasks[1], tasks[2], tasks[3], tasks[4]}
	assertEqualToAllocate(t, toAllocate, []*mockTask{})
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)

	BatchRemove(t, toRelease, expectedToRelease, taskList, false)

	toAllocate, toRelease = p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToRelease(t, taskList, toRelease, []*mockTask{})
	assertEqualToAllocate(t, toAllocate, []*mockTask{newTasks[0], newTasks[1]})
	assert.Assert(t, taskList.len() == 8)
}

func TestPrioritySchedulingPreemptNone(t *testing.T) {
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
		{id: "task1", slotsNeeded: 1, group: groups[1]},
		{id: "task2", slotsNeeded: 1, group: groups[1]},
		{id: "task3", slotsNeeded: 1, group: groups[1]},
		{id: "task4", slotsNeeded: 1, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[0], tasks[1], tasks[2], tasks[3]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task5", slotsNeeded: 2, group: groups[0]},
		{id: "task6", slotsNeeded: 2, group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, []*mockTask{})
	assertEqualToRelease(t, taskList, toRelease, []*mockTask{})
	assert.Assert(t, taskList.len() == 6)
}

func TestPrioritySchedulingPreemptMultiplePriorities(t *testing.T) {
	lowerPriority := 50
	mediumPriority := 40
	higherPriority := 30
	highestPriority := 20

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
		{id: "agent2", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &mediumPriority},
		{id: "group3", priority: &higherPriority},
		{id: "group4", priority: &highestPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[1]},
		{id: "task2", slotsNeeded: 1, group: groups[0]},
		{id: "task3", slotsNeeded: 1, group: groups[0]},
		{id: "task4", slotsNeeded: 1, group: groups[0]},
		{id: "task5", slotsNeeded: 1, group: groups[0]},
		{id: "task6", slotsNeeded: 0, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[0], tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 4, group: groups[2]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	expectedToRelease := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4]}
	assertEqualToAllocate(t, toAllocate, []*mockTask{})
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)

	BatchRemove(t, toRelease, expectedToRelease, taskList, false)

	toAllocate, toRelease = p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToRelease(t, taskList, toRelease, []*mockTask{})
	assertEqualToAllocate(t, toAllocate, []*mockTask{newTasks[0]})
	AllocateTasks(toAllocate, agentMap, taskList)

	priorityTasks := []*mockTask{
		{id: "task8", slotsNeeded: 4, group: groups[3]},
		{id: "task9", slotsNeeded: 4, group: groups[3]},
	}
	AddUnallocatedTasks(t, priorityTasks, system, taskList)

	toAllocate, toRelease = p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	expectedToRelease = []*mockTask{tasks[0], newTasks[0]}
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
	assertEqualToAllocate(t, toAllocate, []*mockTask{})

	BatchRemove(t, toRelease, expectedToRelease, taskList, false)

	toAllocate, toRelease = p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToRelease(t, taskList, toRelease, []*mockTask{})
	assertEqualToAllocate(t, toAllocate, []*mockTask{priorityTasks[0], priorityTasks[1]})
}

func AllocateTasks(
	toAllocate []*AllocateRequest,
	agents map[*actor.Ref]*agentState,
	taskList *taskList,
) {
	for _, req := range toAllocate {
		fits := findFits(req, agents, BestFit)

		for _, fit := range fits {
			container := newContainer(req, fit.Agent, fit.Slots)
			devices := fit.Agent.allocateFreeDevices(fit.Slots, "priority_test")
			allocated := &ResourcesAllocated{
				ID: req.ID,
				Allocations: []Allocation{
					&containerAllocation{
						req:       req,
						agent:     fit.Agent,
						container: container,
						devices:   devices,
					},
				},
			}
			taskList.SetAllocations(req.TaskActor, allocated)
		}
	}
}

func AddUnallocatedTasks(
	t *testing.T,
	mockTasks []*mockTask,
	system *actor.System,
	taskList *taskList,
) {
	for _, mockTask := range mockTasks {
		ref, created := system.ActorOf(actor.Addr(mockTask.id), mockTask)
		assert.Assert(t, created)

		req := &AllocateRequest{
			ID:             mockTask.id,
			SlotsNeeded:    mockTask.slotsNeeded,
			Label:          mockTask.label,
			TaskActor:      ref,
			NonPreemptible: mockTask.nonPreemptible,
		}
		groupRef, _ := system.ActorOf(actor.Addr(mockTask.group.id), mockTask.group)
		req.Group = groupRef

		taskList.AddTask(req)
	}
}

func RemoveTask(slots int, toRelease *actor.Ref, taskList *taskList, delete bool) bool {
	for _, alloc := range taskList.GetAllocations(toRelease).Allocations {
		alloc, ok := alloc.(*containerAllocation)
		if !ok {
			return false
		}
		alloc.agent.deallocateDevices(slots, "priority_test")
	}
	if delete {
		taskList.RemoveTaskByHandler(toRelease)
	} else {
		taskList.RemoveAllocations(toRelease)
	}
	return true
}

func BatchRemove(t *testing.T,
	toRelease []*actor.Ref,
	expectedToRelease []*mockTask,
	taskList *taskList,
	delete bool,
) {
	taskToSlots := map[TaskID]int{}
	for _, task := range expectedToRelease {
		taskToSlots[task.id] = task.slotsNeeded
	}

	for _, taskToRelease := range toRelease {
		task, ok := taskList.GetTaskByHandler(taskToRelease)
		if !ok {
			t.Errorf("Job %s should exist, but doesn't", taskToRelease.Address())
		}
		ok = RemoveTask(taskToSlots[task.ID], taskToRelease, taskList, delete)
		if !ok {
			t.Errorf("Failed to remove task %s", task.ID)
		}
	}
}