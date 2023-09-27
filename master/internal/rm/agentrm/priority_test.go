package agentrm

import (
	"testing"
	"time"

	"github.com/determined-ai/determined/master/internal/rm/tasklist"

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

	agents := make([]*MockAgent, 0)
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, Group: groups[0], JobSubmissionTime: timeNow},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0], JobSubmissionTime: olderTime},
		{ID: "task3", SlotsNeeded: 0, Group: groups[1], JobSubmissionTime: timeNow},
		{ID: "task4", SlotsNeeded: 0, Group: groups[1], JobSubmissionTime: olderTime},
		{ID: "task5", SlotsNeeded: 4, Group: groups[1], JobSubmissionTime: timeNow},
		{ID: "task6", SlotsNeeded: 4, Group: groups[1], JobSubmissionTime: olderTime},
	}

	emptyQueuePositions := make(map[model.JobID]decimal.Decimal)

	system := actor.NewSystem(t.Name())
	taskList, mockGroups, _ := setupSchedulerStates(t, system, tasks, groups, agents)

	zeroSlotPendingTasksByPriority, _ := sortTasksByPriorityAndPositionAndTimestamp(
		taskList, mockGroups, emptyQueuePositions, taskFilter(true))

	tasksInLowerPriority := zeroSlotPendingTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority := []*MockTask{}
	assertEqualToAllocateOrdered(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority := zeroSlotPendingTasksByPriority[higherPriority]
	expectedTasksInHigherPriority := []*MockTask{tasks[3], tasks[2]}
	assertEqualToAllocateOrdered(t, tasksInHigherPriority, expectedTasksInHigherPriority)

	nonZeroSlotPendingTasksByPriority, _ := sortTasksByPriorityAndPositionAndTimestamp(
		taskList, mockGroups, emptyQueuePositions, taskFilter(false))

	tasksInLowerPriority = nonZeroSlotPendingTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority = []*MockTask{tasks[1], tasks[0]}
	assertEqualToAllocateOrdered(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority = nonZeroSlotPendingTasksByPriority[higherPriority]
	expectedTasksInHigherPriority = []*MockTask{tasks[5], tasks[4]}
	assertEqualToAllocateOrdered(t, tasksInHigherPriority, expectedTasksInHigherPriority)

	forceSetTaskAllocations(t, taskList, "task5", 1)
	_, scheduledTasksByPriority := sortTasksByPriorityAndPositionAndTimestamp(
		taskList, mockGroups, emptyQueuePositions, taskFilter(false))

	tasksInLowerPriority = scheduledTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority = make([]*MockTask, 0)
	assertEqualToAllocateOrdered(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority = scheduledTasksByPriority[higherPriority]
	expectedTasksInHigherPriority = []*MockTask{tasks[4]}
	assertEqualToAllocateOrdered(t, tasksInHigherPriority, expectedTasksInHigherPriority)
}

func TestPrioritySchedulingMaxZeroSlotContainer(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4, MaxZeroSlotContainers: 0},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, Group: groups[1]},
		{ID: "task6", SlotsNeeded: 0, Group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*MockTask{tasks[0]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabled(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4, MaxZeroSlotContainers: 100},
		{ID: "agent2", Slots: 4, MaxZeroSlotContainers: 100},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task3", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task4", SlotsNeeded: 0, Group: groups[1]},
		{ID: "task5", SlotsNeeded: 4, Group: groups[1]},
		{ID: "task6", SlotsNeeded: 0, Group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*MockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	// Check that agent stat has not changed.
	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}
}

func TestPrioritySchedulingPreemptionDisabledHigherPriorityBlocksLowerPriority(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task3", SlotsNeeded: 12, Group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*MockTask{}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	// Check that agent stat has not changed.
	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}
}

func TestPrioritySchedulingPreemptionDisabledAddTasks(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4, MaxZeroSlotContainers: 100},
		{ID: "agent2", Slots: 4, MaxZeroSlotContainers: 100},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task3", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task4", SlotsNeeded: 0, Group: groups[1]},
		{ID: "task5", SlotsNeeded: 4, Group: groups[1]},
		{ID: "task6", SlotsNeeded: 0, Group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*MockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*MockTask{
		{ID: "task7", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task8", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task9", SlotsNeeded: 1, Group: groups[0]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, _ = p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*MockTask{newTasks[0], newTasks[1]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledAllSlotsAllocated(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task3", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task4", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task5", SlotsNeeded: 4, Group: groups[1]},
		{ID: "task6", SlotsNeeded: 1, Group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*MockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*MockTask{
		{ID: "task7", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task8", SlotsNeeded: 1, Group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, _ = p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*MockTask{}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledLowerPriorityMustWait(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task3", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task4", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task5", SlotsNeeded: 2, Group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	firstAllocation, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*MockTask{tasks[1], tasks[2], tasks[3]}
	assertEqualToAllocate(t, firstAllocation, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(firstAllocation, agentMap, taskList)

	secondAllocation, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*MockTask{}
	assertEqualToAllocate(t, secondAllocation, expectedToAllocate)

	for _, task := range firstAllocation {
		RemoveTask(task.SlotsNeeded, task.AllocationID, taskList, true)
	}

	thirdAllocation, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*MockTask{tasks[0], tasks[4]}
	assertEqualToAllocate(t, thirdAllocation, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledTaskFinished(t *testing.T) {
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4, MaxZeroSlotContainers: 100},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, Group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)
	ok := RemoveTask(4, toAllocate[0].AllocationID, taskList, true)
	if !ok {
		t.Errorf("Failed to remove task %s", toAllocate[0].AllocationID)
	}

	newTasks := []*MockTask{
		{ID: "task7", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task8", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task9", SlotsNeeded: 0, Group: groups[0]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, _ = p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate := []*MockTask{newTasks[0], newTasks[1], newTasks[2]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledAllTasksFinished(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task3", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task4", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task5", SlotsNeeded: 4, Group: groups[1]},
		{ID: "task6", SlotsNeeded: 1, Group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*MockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*MockTask{
		{ID: "task7", SlotsNeeded: 4, Group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	for _, task := range toAllocate {
		RemoveTask(task.SlotsNeeded, task.AllocationID, taskList, true)
	}

	toAllocate, _ = p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedToAllocate = []*MockTask{tasks[0], newTasks[0]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionDisabledZeroSlotTask(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4, MaxZeroSlotContainers: 1},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 0, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 0, Group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)

	expectedToAllocate := []*MockTask{tasks[0]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	AllocateTasks(toAllocate, agentMap, taskList)

	newTasks := []*MockTask{
		{ID: "task3", SlotsNeeded: 0, Group: groups[1]},
	}
	AddUnallocatedTasks(t, newTasks, system, taskList)

	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	expectedTasks := []*MockTask{}
	assertEqualToAllocate(t, toAllocate, expectedTasks)
	assertEqualToRelease(t, taskList, toRelease, expectedTasks)
}

func TestPrioritySchedulingPreemption(t *testing.T) {
	lowerPriority := 50
	mediumPriority := 45
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &mediumPriority},
		{ID: "group3", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{
			ID:          "low-priority task cannot be backfilled because preemption exists",
			SlotsNeeded: 1, Group: groups[0],
		},
		{
			ID:          "medium-priority task should be preempted",
			SlotsNeeded: 4, Group: groups[1], AllocatedAgent: agents[0], ContainerStarted: true,
		},
		{
			ID:          "high-priority task should not be preempted",
			SlotsNeeded: 4, Group: groups[2], AllocatedAgent: agents[1], ContainerStarted: true,
		},
		{
			ID:          "high-priority task causes preemption but should not be scheduled",
			SlotsNeeded: 4, Group: groups[2],
		},
		{
			ID:          "high-priority oversized task triggers backfilling",
			SlotsNeeded: 8, Group: groups[2],
		},
	}

	expectedToAllocate := []*MockTask{}
	expectedToRelease := []*MockTask{tasks[1]}

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

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowestPriority},
		{ID: "group2", Priority: &lowerPriority},
		{ID: "group3", Priority: &mediumPriority},
		{ID: "group4", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{
			ID:          "low-priority task should be preempted",
			SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0], ContainerStarted: true,
		},
		{
			ID:          "lower-priority task causes preemption but should not be scheduled",
			SlotsNeeded: 1, Group: groups[1],
		},
		{
			ID:          "medium-priority task should be backfilled",
			SlotsNeeded: 1, Group: groups[2],
		},
		{
			ID:          "high-priority task should not be preempted",
			SlotsNeeded: 4, Group: groups[3], AllocatedAgent: agents[1], ContainerStarted: true,
		},
		{
			ID:          "high-priority task should be scheduled",
			SlotsNeeded: 2, Group: groups[3],
		},
		{
			ID:          "high-priority oversized task triggers backfilling",
			SlotsNeeded: 8, Group: groups[3],
		},
	}

	expectedToAllocate := []*MockTask{tasks[2], tasks[4]}
	expectedToRelease := []*MockTask{tasks[0]}

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

	agents := []*MockAgent{
		{ID: "agent1", MaxZeroSlotContainers: 1},
		{ID: "agent2", MaxZeroSlotContainers: 1},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority},
		{ID: "group2", Priority: &mediumPriority},
		{ID: "group3", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{
			ID:          "low-priority task cannot be scheduled",
			SlotsNeeded: 0, Group: groups[0],
		},
		{
			ID:          "medium-priority task should be preempted",
			SlotsNeeded: 0, Group: groups[1], AllocatedAgent: agents[0], ContainerStarted: true,
		},
		{
			ID:          "high-priority task should not be preempted",
			SlotsNeeded: 0, Group: groups[2], AllocatedAgent: agents[1], ContainerStarted: true,
		},
		{
			ID:          "high-priority task causes preemption but should not be scheduled",
			SlotsNeeded: 0, Group: groups[2],
		},
	}

	expectedToAllocate := []*MockTask{}
	expectedToRelease := []*MockTask{tasks[1]}

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

	agents := []*MockAgent{
		{ID: "agent1", MaxZeroSlotContainers: 4},
		{ID: "agent2", MaxZeroSlotContainers: 1},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowestPriority},
		{ID: "group2", Priority: &lowerPriority},
		{ID: "group3", Priority: &mediumPriority},
		{ID: "group4", Priority: &higherPriority},
	}
	tasks := []*MockTask{
		{
			ID:          "low-priority task should be scheduled",
			SlotsNeeded: 0, Group: groups[0],
		},
		{
			ID:          "medium-priority task should not be preempted",
			SlotsNeeded: 0, Group: groups[1], AllocatedAgent: agents[0], ContainerStarted: true,
		},
		{
			ID:          "high-priority task should not be preempted",
			SlotsNeeded: 0, Group: groups[2], AllocatedAgent: agents[1], ContainerStarted: true,
		},
		{
			ID:          "high-priority task should be scheduled",
			SlotsNeeded: 0, Group: groups[2],
		},
	}

	expectedToAllocate := []*MockTask{tasks[0], tasks[3]}
	expectedToRelease := []*MockTask{}

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

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "1", Priority: &priority},
		{ID: "2", Priority: &priority},
		{ID: "3", Priority: &priority},
	}
	tasks := []*MockTask{
		{
			ID:          "1",
			JobID:       "1",
			SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0], ContainerStarted: true,
		},
		{
			ID:          "2",
			JobID:       "2",
			SlotsNeeded: 4, Group: groups[1], AllocatedAgent: agents[1], ContainerStarted: true,
		},
		{
			ID:          "3",
			JobID:       "3",
			SlotsNeeded: 4, Group: groups[2],
		},
	}

	expectedToAllocate := []*MockTask{}
	expectedToRelease := []*MockTask{tasks[1]}
	jobsList := map[model.JobID]decimal.Decimal{
		"1": decimal.New(1, sproto.DecimalExp),
		"2": decimal.New(2, sproto.DecimalExp),
		"3": decimal.NewFromFloatWithExponent(1.5, sproto.DecimalExp),
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, jobsList, agentMap, BestFit)
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

	agents := []*MockAgent{
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &priority},
		{ID: "group2", Priority: &priority},
		{ID: "group3", Priority: &priority},
	}
	tasks := []*MockTask{
		{
			ID:          "1",
			JobID:       "1",
			SlotsNeeded: 1, Group: groups[0], AllocatedAgent: agents[0], ContainerStarted: true,
		},
		{
			ID:          "2",
			JobID:       "2",
			SlotsNeeded: 4, Group: groups[1], AllocatedAgent: agents[1], ContainerStarted: true,
		},
		{
			ID:          "3",
			JobID:       "3",
			SlotsNeeded: 8, Group: groups[2],
		},
	}

	expectedToAllocate := []*MockTask{}
	expectedToRelease := []*MockTask{}
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
	agents map[*actor.Ref]*agentState,
	taskList *tasklist.TaskList,
) {
	for _, req := range toAllocate {
		fits := findFits(req, agents, BestFit, false)

		for _, fit := range fits {
			containerID := cproto.NewID()
			devices, err := fit.Agent.allocateFreeDevices(fit.Slots, containerID)
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
			taskList.AddAllocation(req.AllocationID, allocated)
		}
	}
}

func AddUnallocatedTasks(
	t *testing.T,
	mockTasks []*MockTask,
	system *actor.System,
	taskList *tasklist.TaskList,
) {
	for _, mockTask := range mockTasks {
		ref, created := system.ActorOf(actor.Addr(mockTask.ID), mockTask)
		assert.Assert(t, created)
		req := MockTaskToAllocateRequest(mockTask, ref)
		if mockTask.Group != nil {
			req.JobID = model.JobID(mockTask.Group.ID)
		}

		taskList.AddTask(req)
	}
}

func RemoveTask(
	slots int,
	toRelease model.AllocationID,
	taskList *tasklist.TaskList,
	delete bool,
) bool {
	for _, alloc := range taskList.Allocation(toRelease).Resources {
		alloc, ok := alloc.(*containerResources)
		if !ok {
			return false
		}
		alloc.agent.deallocateContainer(alloc.containerID)
	}
	if delete {
		taskList.RemoveTaskByID(toRelease)
	} else {
		taskList.RemoveAllocation(toRelease)
	}
	return true
}
