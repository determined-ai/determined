package agentrm

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestRoundRobinScheduler(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent1", Slots: 4, MaxZeroSlotContainers: 100},
		{ID: "agent2", Slots: 4, MaxZeroSlotContainers: 100},
		{ID: "agent3", Slots: 4, MaxZeroSlotContainers: 100},
	}
	groups := []*MockGroup{
		{ID: "group1", MaxSlots: newMaxSlot(1), Weight: 1},
	}
	tasks := []*MockTask{
		{ID: "task1", SlotsNeeded: 4, Group: groups[0]},
		{ID: "task2", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task3", SlotsNeeded: 0, Group: groups[0]},
	}

	expectedToAllocate := []*MockTask{tasks[0], tasks[1], tasks[2]}
	expectedToRelease := []*MockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, toRelease := roundRobinSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}
