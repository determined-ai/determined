//go:build integration
// +build integration

package agentrm

import (
	"testing"

	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/set"
)

func TestFairShareBlocklist(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent", Slots: 1},
	}
	groups := []*MockGroup{
		{ID: "group0"},
		{ID: "group1"},
	}
	tasks := []*MockTask{
		{ID: "task0.1", TaskID: "task0", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task1.1", TaskID: "task1", SlotsNeeded: 1, Group: groups[1]},
	}

	logpattern.SetDisallowedNodesCacheTest(t, map[model.TaskID]*set.Set[string]{
		"task0": ptrs.Ptr(set.FromSlice([]string{"agent"})),
	})

	expectedToAllocate := []*MockTask{tasks[1]}
	expectedToRelease := []*MockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareBlocklistMultiple(t *testing.T) {
	// This is kind of a strange situation occurring here
	// We have 4 agents but 2 of the agents are disabled by this task.
	// So we are going to request to allocate both of these tasks even though
	// we know we can't schedule these both. allocateResources should handle
	// this case and only let us schedule one of these at a time.
	agents := []*MockAgent{
		{ID: "agent0", Slots: 4},
		{ID: "agent1", Slots: 4},
		{ID: "agent2", Slots: 4},
		{ID: "agent3", Slots: 4},
	}
	groups := []*MockGroup{
		{ID: "group0"},
		{ID: "group1"},
	}
	tasks := []*MockTask{
		{ID: "task0.1", TaskID: "task0", SlotsNeeded: 8, Group: groups[0]},
		{ID: "task1.1", TaskID: "task1", SlotsNeeded: 8, Group: groups[1]},
	}

	logpattern.SetDisallowedNodesCacheTest(t, map[model.TaskID]*set.Set[string]{
		"task0": ptrs.Ptr(set.FromSlice([]string{"agent2", "agent3"})),
		"task1": ptrs.Ptr(set.FromSlice([]string{"agent2", "agent3"})),
	})

	expectedToAllocate := []*MockTask{tasks[0], tasks[1]}
	expectedToRelease := []*MockTask{}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareBlocklistPreemptible(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent0", Slots: 1},
		{ID: "agent1", Slots: 1},
	}
	tasks := []*MockTask{
		{ID: "task0.1", TaskID: "task0", SlotsNeeded: 1},
		{ID: "task1.1", TaskID: "task1", SlotsNeeded: 1, AllocatedAgent: agents[0]},
		{ID: "task2.1", TaskID: "task2", SlotsNeeded: 1},
		{ID: "task3.1", TaskID: "task3", SlotsNeeded: 1, AllocatedAgent: agents[1]},
	}

	expectedToAllocate := []*MockTask{tasks[2]}
	expectedToRelease := []*MockTask{tasks[3]}

	logpattern.SetDisallowedNodesCacheTest(t, map[model.TaskID]*set.Set[string]{
		"task0": ptrs.Ptr(set.FromSlice([]string{"agent0", "agent1"})),
		"task2": ptrs.Ptr(set.FromSlice([]string{"agent1"})),
		// This is also a weird case. We preempt a running job to try and schedule our job
		// which can't schedule since it is blocked on agent1.
	})

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, nil, agents)

	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestFairShareBlocklistDontPreempt(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent0", Slots: 1},
	}
	tasks := []*MockTask{
		{ID: "task0.1", TaskID: "task0", SlotsNeeded: 1},
		{ID: "task1.1", TaskID: "task1", SlotsNeeded: 1},
		{ID: "task2.1", TaskID: "task2", SlotsNeeded: 1},
		{ID: "task3.1", TaskID: "task3", SlotsNeeded: 1, AllocatedAgent: agents[0]},
	}

	expectedToAllocate := []*MockTask{}
	expectedToRelease := []*MockTask{}

	logpattern.SetDisallowedNodesCacheTest(t, map[model.TaskID]*set.Set[string]{
		"task0": ptrs.Ptr(set.FromSlice([]string{"agent0"})),
		"task1": ptrs.Ptr(set.FromSlice([]string{"agent0"})),
		"task2": ptrs.Ptr(set.FromSlice([]string{"agent0"})),
	})

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, nil, agents)

	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}
