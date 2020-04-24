package scheduler

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestFairShare_Schedule_MaxSlots(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, "agent", 4, "")}

	group1 := newCustomGroup(t, system, "group1", 1, 1)
	group2 := newGroup(t, system, "group2")

	tasks := []*actor.Ref{
		newMockTask(t, system, group1, "task1", 1, ""),
		newMockTask(t, system, group1, "task2", 1, ""),
		newMockTask(t, system, group1, "task3", 1, ""),
		newMockTask(t, system, group1, "task4", 1, ""),

		newMockTask(t, system, group2, "task5", 1, ""),
		newMockTask(t, system, group2, "task6", 1, ""),
		newMockTask(t, system, group2, "task7", 1, ""),
		newMockTask(t, system, group2, "task8", 1, ""),
	}

	expected := []schedulerState{
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskPending},
		{state: taskPending},
		{state: taskPending},

		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskPending},
	}

	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}

func TestFairShare_Schedule_Weights(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, "agent", 8, "")}

	group1 := newCustomGroup(t, system, "group1", 100, 10)
	group2 := newCustomGroup(t, system, "group2", 100, 30)

	tasks := []*actor.Ref{
		newMockTask(t, system, group1, "task1", 1, ""),
		newMockTask(t, system, group1, "task2", 1, ""),
		newMockTask(t, system, group1, "task3", 1, ""),

		newMockTask(t, system, group2, "task4", 1, ""),
		newMockTask(t, system, group2, "task5", 1, ""),
		newMockTask(t, system, group2, "task6", 1, ""),
		newMockTask(t, system, group2, "task7", 1, ""),
		newMockTask(t, system, group2, "task8", 1, ""),
		newMockTask(t, system, group2, "task9", 1, ""),
		newMockTask(t, system, group2, "task10", 1, ""),
	}

	expected := []schedulerState{
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskPending},

		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskPending},
	}

	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}

func TestFairShare_Schedule_MultiSlot(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{
		newMockAgent(t, system, "agent1", 4, ""),
		newMockAgent(t, system, "agent2", 4, ""),
	}

	group1 := newGroup(t, system, "group1")
	group2 := newGroup(t, system, "group2")

	tasks := []*actor.Ref{
		newMockTask(t, system, group1, "task1", 4, ""),
		newMockTask(t, system, group2, "task2", 4, ""),
	}

	expected := []schedulerState{
		{state: taskRunning, containers: map[*agentState]int{agents[1]: 4}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 4}},
	}

	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}

func TestFairShare_Schedule_MaxSlotsStartingTasks(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, "agent", 4, "")}

	group1 := newCustomGroup(t, system, "group1", 2, 1)

	tasks := []*actor.Ref{
		newMockTask(t, system, group1, "task1", 1, ""),
		newMockTask(t, system, group1, "task2", 1, ""),
		newMockTask(t, system, group1, "task3", 1, ""),
		newMockTask(t, system, group1, "task4", 1, ""),
	}

	expected := []schedulerState{
		{state: taskTerminating, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskTerminating, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
	}

	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)

	forceSchedule(c, tasks[0], agents[0])
	forceSchedule(c, tasks[1], agents[0])
	forceSchedule(c, tasks[2], agents[0])
	forceSchedule(c, tasks[3], agents[0])

	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}

func TestFairShare_Schedule_UnscheduledNewTasks(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{
		newMockAgent(t, system, "agent1", 2, ""),
		newMockAgent(t, system, "agent2", 2, ""),
	}

	group1 := newCustomGroup(t, system, "group1", 2, 1)

	tasks := []*actor.Ref{
		newMockTask(t, system, group1, "task1", 2, ""),
		newMockTask(t, system, group1, "task2", 1, ""),
		newMockTask(t, system, group1, "task3", 1, ""),
	}

	expected := []schedulerState{
		{state: taskPending},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[1]: 1}},
	}

	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)

	forceSchedule(c, tasks[1], agents[0])
	forceSchedule(c, tasks[2], agents[1])

	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}

func TestFairShare_Schedule_MultiSlotDeadlock(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, "agent", 2, "")}

	group1 := newGroup(t, system, "group1")
	group2 := newGroup(t, system, "group2")

	tasks := []*actor.Ref{
		newMockTask(t, system, group1, "task1", 2, ""),
		newMockTask(t, system, group2, "task2", 2, ""),
	}

	expected := []schedulerState{
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 2}},
		{state: taskPending},
	}

	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}

// Test that a single task with slot demand higher than the cluster capacity does not prevent other
// tasks from being scheduled.
func TestFairShare_Schedule_BigTask(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, "agent", 4, "")}

	group1 := newGroup(t, system, "group1")
	group2 := newGroup(t, system, "group2")

	tasks := []*actor.Ref{
		newMockTask(t, system, group1, "task1", 5, ""),
		newMockTask(t, system, group2, "task2", 4, ""),
	}

	expected := []schedulerState{
		{state: taskPending},
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 4}},
	}

	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}

func TestFairShare_Schedule_ActiveTasks(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{
		newMockAgent(t, system, "agent1", 4, ""),
		newMockAgent(t, system, "agent2", 3, ""),
	}

	group1 := newGroup(t, system, "group1")
	group2 := newGroup(t, system, "group2")
	group3 := newGroup(t, system, "group3")
	group4 := newGroup(t, system, "group4")

	tasks := []*actor.Ref{
		newMockTask(t, system, group1, "task1", 3, ""),
		newMockTask(t, system, group2, "task2", 1, ""),
		newMockTask(t, system, group2, "task3", 1, ""),
		newMockTask(t, system, group3, "task4", 4, ""),
		newMockTask(t, system, group4, "task5", 1, ""),
	}

	expected := []schedulerState{
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 3}},
		{state: taskRunning, containers: map[*agentState]int{agents[1]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[1]: 1}},
		{state: taskPending},
		{state: taskRunning, containers: map[*agentState]int{agents[1]: 1}},
	}

	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)

	forceSchedule(c, tasks[2], agents[1])

	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}

func TestFairShare_ScheduleNilGroup(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, "agent", 4, "")}

	tasks := []*actor.Ref{
		newMockTask(t, system, nil, "task1", 4, ""),
		newMockTask(t, system, nil, "task2", 1, ""),
	}

	expected := []schedulerState{
		{state: taskRunning, containers: map[*agentState]int{agents[0]: 4}},
		{state: taskPending},
	}

	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}

func TestFairShare_Schedule_Labels(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{
		newMockAgent(t, system, "agent-1", 4, ""),
		newMockAgent(t, system, "agent-2", 4, "label-1"),
		newMockAgent(t, system, "agent-3", 4, "label-2"),
	}

	group1 := newCustomGroup(t, system, "group1", 1, 1)

	tasks := []*actor.Ref{
		newMockTask(t, system, group1, "task1", 4, "label-1"),
		newMockTask(t, system, group1, "task2", 1, "label-2"),
		newMockTask(t, system, group1, "task3", 0, "label-2"),
	}

	expected := []schedulerState{
		{state: taskRunning, containers: map[*agentState]int{agents[1]: 4}},
		{state: taskRunning, containers: map[*agentState]int{agents[2]: 1}},
		{state: taskRunning, containers: map[*agentState]int{agents[2]: 0}},
	}

	c := setupCluster(NewPriorityScheduler(), BestFit, agents, tasks)
	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}
