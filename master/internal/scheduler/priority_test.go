package scheduler

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestPriority_Schedule_Labels(t *testing.T) {
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
		{containers: map[*agentState]int{agents[1]: 4}},
		{containers: map[*agentState]int{agents[2]: 1}},
		{containers: map[*agentState]int{agents[2]: 0}},
	}

	c := setupCluster(NewPriorityScheduler(), BestFit, agents, tasks)
	c.scheduler.Schedule(c)
	assertSchedulerState(t, c, tasks, expected)
}
