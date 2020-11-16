package resourcemanagers

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestRoundRobinSchedulerLabels(t *testing.T) {
	agents := []*mockAgent{
		{id: "agent1", slots: 4, label: ""},
		{id: "agent2", slots: 4, label: "label1"},
		{id: "agent3", slots: 4, label: "label2"},
	}
	groups := []*mockGroup{
		{id: "group1", maxSlots: newMaxSlot(1), weight: 1},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0], label: "label1"},
		{id: "task2", slotsNeeded: 1, group: groups[0], label: "label2"},
		{id: "task3", slotsNeeded: 0, group: groups[0], label: "label2"},
	}

	expectedToAllocate := []*mockTask{tasks[0], tasks[1], tasks[2]}
	expectedToRelease := []*mockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := roundRobinSchedule(taskList, groupMap, agentMap, BestFit)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}
