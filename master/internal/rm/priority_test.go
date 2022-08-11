package rm

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestSortTasksByPriorityAndTimestamps(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	timeNow := time.Now()
	olderTime := timeNow.Add(-time.Minute * 15)

	agents := make([]*mockAgent, 0)
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0], jobSubmissionTime: timeNow},
		{id: "task2", slotsNeeded: 1, group: groups[0], jobSubmissionTime: olderTime},
		{id: "task3", slotsNeeded: 0, group: groups[1], jobSubmissionTime: timeNow},
		{id: "task4", slotsNeeded: 0, group: groups[1], jobSubmissionTime: olderTime},
		{id: "task5", slotsNeeded: 4, group: groups[1], jobSubmissionTime: timeNow},
		{id: "task6", slotsNeeded: 4, group: groups[1], jobSubmissionTime: olderTime},
	}

	emptyQueuePositions := make(map[model.JobID]decimal.Decimal)

	system := actor.NewSystem(t.Name())
	taskList, mockGroups, _ := setupSchedulerStates(t, system, tasks, groups, agents)

	zeroSlotPendingTasksByPriority, _ := sortTasksByPriorityAndPositionAndTimestamp(
		taskList, mockGroups, emptyQueuePositions, taskFilter("", true))

	tasksInLowerPriority := zeroSlotPendingTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority := []*mockTask{}
	assertEqualToAllocateOrdered(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority := zeroSlotPendingTasksByPriority[higherPriority]
	expectedTasksInHigherPriority := []*mockTask{tasks[3], tasks[2]}
	assertEqualToAllocateOrdered(t, tasksInHigherPriority, expectedTasksInHigherPriority)

	nonZeroSlotPendingTasksByPriority, _ := sortTasksByPriorityAndPositionAndTimestamp(
		taskList, mockGroups, emptyQueuePositions, taskFilter("", false))

	tasksInLowerPriority = nonZeroSlotPendingTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority = []*mockTask{tasks[1], tasks[0]}
	assertEqualToAllocateOrdered(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority = nonZeroSlotPendingTasksByPriority[higherPriority]
	expectedTasksInHigherPriority = []*mockTask{tasks[5], tasks[4]}
	assertEqualToAllocateOrdered(t, tasksInHigherPriority, expectedTasksInHigherPriority)

	forceSetTaskAllocations(t, taskList, "task5", 1)
	_, scheduledTasksByPriority := sortTasksByPriorityAndPositionAndTimestamp(
		taskList, mockGroups, emptyQueuePositions, taskFilter("", false))

	tasksInLowerPriority = scheduledTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority = make([]*mockTask, 0)
	assertEqualToAllocateOrdered(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority = scheduledTasksByPriority[higherPriority]
	expectedTasksInHigherPriority = []*mockTask{tasks[4]}
	assertEqualToAllocateOrdered(t, tasksInHigherPriority, expectedTasksInHigherPriority)
}

func TestPrioritySchedulingMaxZeroSlotContainer(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 0},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[1]},
		{id: "task6", slotsNeeded: 0, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[0]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
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
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	// Check that agent stat has not changed.
	for _, agent := range agentMap {
		assert.Equal(t, agent.NumEmptySlots(), 4)
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
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*mockTask{}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	// Check that agent stat has not changed.
	for _, agent := range agentMap {
		assert.Equal(t, agent.NumEmptySlots(), 4)
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
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[0], tasks[1]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledAddTasks(t *testing.T) {
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
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.NumEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 1, group: groups[0]},
		{id: "task8", slotsNeeded: 1, group: groups[0]},
		{id: "task9", slotsNeeded: 1, group: groups[0]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, _ = p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*mockTask{newTasks[0], newTasks[1]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledAllSlotsAllocated(t *testing.T) {
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
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.NumEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 1, group: groups[1]},
		{id: "task8", slotsNeeded: 1, group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, _ = p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*mockTask{}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledLowerPriorityMustWait(t *testing.T) {
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
	firstAllocation, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3]}
	assertEqualToAllocate(t, firstAllocation, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.NumEmptySlots(), 4)
	}

	AllocateTasks(firstAllocation, agentMap, taskList)

	secondAllocation, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*mockTask{}
	assertEqualToAllocate(t, secondAllocation, expectedToAllocate)

	for _, task := range firstAllocation {
		RemoveTask(task.SlotsNeeded, task.AllocationRef, taskList, true)
	}

	thirdAllocation, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*mockTask{tasks[0], tasks[4]}
	assertEqualToAllocate(t, thirdAllocation, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledTaskFinished(t *testing.T) {
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 100},
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
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	for _, agent := range agentMap {
		assert.Equal(t, agent.NumEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)
	ok := RemoveTask(4, toAllocate[0].AllocationRef, taskList, true)
	if !ok {
		t.Errorf("Failed to remove task %s", toAllocate[0].AllocationID)
	}

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 1, group: groups[0]},
		{id: "task8", slotsNeeded: 1, group: groups[0]},
		{id: "task9", slotsNeeded: 0, group: groups[0]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, _ = p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate := []*mockTask{newTasks[0], newTasks[1], newTasks[2]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledAllTasksFinished(t *testing.T) {
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
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.NumEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task7", slotsNeeded: 4, group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	for _, task := range toAllocate {
		RemoveTask(task.SlotsNeeded, task.AllocationRef, taskList, true)
	}

	toAllocate, _ = p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*mockTask{tasks[0], newTasks[0]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledZeroSlotTask(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 1},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 0, group: groups[0]},
		{id: "task2", slotsNeeded: 0, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[0]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*mockTask{
		{id: "task3", slotsNeeded: 0, group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedTasks := []*mockTask{}
	assertEqualToAllocate(t, toAllocate, expectedTasks)
	assertEqualToRelease(t, taskList, toRelease, expectedTasks)
}

func TestPrioritySchedulingPreemption(t *testing.T) {
	lowerPriority := 50
	mediumPriority := 45
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
		{id: "agent2", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &mediumPriority},
		{id: "group3", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{
			id:          "low-priority task cannot be backfilled because preemption exists",
			slotsNeeded: 1, group: groups[0],
		},
		{
			id:          "medium-priority task should be preempted",
			slotsNeeded: 4, group: groups[1], allocatedAgent: agents[0], containerStarted: true,
		},
		{
			id:          "high-priority task should not be preempted",
			slotsNeeded: 4, group: groups[2], allocatedAgent: agents[1], containerStarted: true,
		},
		{
			id:          "high-priority task causes preemption but should not be scheduled",
			slotsNeeded: 4, group: groups[2],
		},
		{
			id:          "high-priority oversized task triggers backfilling",
			slotsNeeded: 8, group: groups[2],
		},
	}

	expectedToAllocate := []*mockTask{}
	expectedToRelease := []*mockTask{tasks[1]}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestPrioritySchedulingBackfilling(t *testing.T) {
	lowestPriority := 55
	lowerPriority := 50
	mediumPriority := 45
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
		{id: "agent2", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowestPriority},
		{id: "group2", priority: &lowerPriority},
		{id: "group3", priority: &mediumPriority},
		{id: "group4", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{
			id:          "low-priority task should be preempted",
			slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0], containerStarted: true,
		},
		{
			id:          "lower-priority task causes preemption but should not be scheduled",
			slotsNeeded: 1, group: groups[1],
		},
		{
			id:          "medium-priority task should be backfilled",
			slotsNeeded: 1, group: groups[2],
		},
		{
			id:          "high-priority task should not be preempted",
			slotsNeeded: 4, group: groups[3], allocatedAgent: agents[1], containerStarted: true,
		},
		{
			id:          "high-priority task should be scheduled",
			slotsNeeded: 2, group: groups[3],
		},
		{
			id:          "high-priority oversized task triggers backfilling",
			slotsNeeded: 8, group: groups[3],
		},
	}

	expectedToAllocate := []*mockTask{tasks[2], tasks[4]}
	expectedToRelease := []*mockTask{tasks[0]}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestPrioritySchedulingPreemptionZeroSlotTask(t *testing.T) {
	lowerPriority := 50
	mediumPriority := 45
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", maxZeroSlotContainers: 1},
		{id: "agent2", maxZeroSlotContainers: 1},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &mediumPriority},
		{id: "group3", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{
			id:          "low-priority task cannot be scheduled",
			slotsNeeded: 0, group: groups[0],
		},
		{
			id:          "medium-priority task should be preempted",
			slotsNeeded: 0, group: groups[1], allocatedAgent: agents[0], containerStarted: true,
		},
		{
			id:          "high-priority task should not be preempted",
			slotsNeeded: 0, group: groups[2], allocatedAgent: agents[1], containerStarted: true,
		},
		{
			id:          "high-priority task causes preemption but should not be scheduled",
			slotsNeeded: 0, group: groups[2],
		},
	}

	expectedToAllocate := []*mockTask{}
	expectedToRelease := []*mockTask{tasks[1]}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestPrioritySchedulingBackfillingZeroSlotTask(t *testing.T) {
	lowestPriority := 55
	lowerPriority := 50
	mediumPriority := 45
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", maxZeroSlotContainers: 4},
		{id: "agent2", maxZeroSlotContainers: 1},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowestPriority},
		{id: "group2", priority: &lowerPriority},
		{id: "group3", priority: &mediumPriority},
		{id: "group4", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{
			id:          "low-priority task should be scheduled",
			slotsNeeded: 0, group: groups[0],
		},
		{
			id:          "medium-priority task should not be preempted",
			slotsNeeded: 0, group: groups[1], allocatedAgent: agents[0], containerStarted: true,
		},
		{
			id:          "high-priority task should not be preempted",
			slotsNeeded: 0, group: groups[2], allocatedAgent: agents[1], containerStarted: true,
		},
		{
			id:          "high-priority task should be scheduled",
			slotsNeeded: 0, group: groups[2],
		},
	}

	expectedToAllocate := []*mockTask{tasks[0], tasks[3]}
	expectedToRelease := []*mockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestPrioritySchedulingPreemptOneByPosition(t *testing.T) {
	priority := 42

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
		{id: "agent2", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &priority},
		{id: "group2", priority: &priority},
		{id: "group3", priority: &priority},
	}
	tasks := []*mockTask{
		{
			id:          "1",
			jobID:       "1",
			slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0], containerStarted: true,
		},
		{
			id:          "2",
			jobID:       "2",
			slotsNeeded: 4, group: groups[1], allocatedAgent: agents[1], containerStarted: true,
		},
		{
			id:          "3",
			jobID:       "3",
			slotsNeeded: 4, group: groups[2],
		},
	}

	expectedToAllocate := []*mockTask{}
	expectedToRelease := []*mockTask{tasks[1]}
	jobsList := map[model.JobID]decimal.Decimal{
		"1": decimal.New(1, sproto.DecimalExp),
		"2": decimal.New(2, sproto.DecimalExp),
		"3": decimal.NewFromFloatWithExponent(1.5, sproto.DecimalExp),
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		jobsList, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)

	// even if the position is higher, still only preempt one
	jobsList = map[model.JobID]decimal.Decimal{
		"1": decimal.New(1, sproto.DecimalExp),
		"2": decimal.New(1, sproto.DecimalExp),
		"3": decimal.NewFromFloatWithExponent(0.999, sproto.DecimalExp),
	}

	toAllocate, toRelease = p.prioritySchedule(taskList, groupMap,
		jobsList, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestPrioritySchedulingNoPreemptionByPosition(t *testing.T) {
	priority := 42

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
		{id: "agent2", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &priority},
		{id: "group2", priority: &priority},
		{id: "group3", priority: &priority},
	}
	tasks := []*mockTask{
		{
			id:          "1",
			jobID:       "1",
			slotsNeeded: 1, group: groups[0], allocatedAgent: agents[0], containerStarted: true,
		},
		{
			id:          "2",
			jobID:       "2",
			slotsNeeded: 4, group: groups[1], allocatedAgent: agents[1], containerStarted: true,
		},
		{
			id:          "3",
			jobID:       "3",
			slotsNeeded: 8, group: groups[2],
		},
	}

	expectedToAllocate := []*mockTask{}
	expectedToRelease := []*mockTask{}
	jobsList := map[model.JobID]decimal.Decimal{
		"1": decimal.New(1, sproto.DecimalExp),
		"2": decimal.New(2, sproto.DecimalExp),
	}
	jobsList["3"] = decimal.Avg(jobsList["1"], jobsList["2"])

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		jobsList, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func AllocateTasks(
	toAllocate []*sproto.AllocateRequest,
	agents map[*actor.Ref]*AgentState,
	taskList *taskList,
) {
	for _, req := range toAllocate {
		fits := findFits(req, agents, BestFit)

		for _, fit := range fits {
			containerID := cproto.NewID()
			devices, err := fit.Agent.AllocateFreeDevices(fit.Slots, containerID)
			if err != nil {
				panic(err)
			}
			allocated := &sproto.ResourcesAllocated{
				ID: req.AllocationID,
				Resources: map[sproto.ResourcesID]sproto.Resources{
					sproto.ResourcesID(containerID): &containerResources{
						req:         req,
						agent:       fit.Agent,
						containerID: containerID,
						devices:     devices,
					},
				},
			}
			taskList.SetAllocations(req.AllocationRef, allocated)
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
		req := mockTaskToAllocateRequest(mockTask, ref)
		if mockTask.group != nil {
			groupRef, _ := system.ActorOf(actor.Addr(mockTask.group.id), mockTask.group)
			req.Group = groupRef
		}

		taskList.AddTask(req)
	}
}

func RemoveTask(slots int, toRelease *actor.Ref, taskList *taskList, delete bool) bool {
	for _, alloc := range taskList.GetAllocations(toRelease).Resources {
		alloc, ok := alloc.(*containerResources)
		if !ok {
			return false
		}
		alloc.agent.DeallocateContainer(alloc.containerID)
	}
	if delete {
		taskList.RemoveTaskByHandler(toRelease)
	} else {
		taskList.RemoveAllocations(toRelease)
	}
	return true
}
