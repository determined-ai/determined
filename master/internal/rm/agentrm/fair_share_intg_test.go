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
		{ID: "group1"},
		{ID: "group2"},
	}
	tasks := []*MockTask{
		{ID: "task1.1", TaskID: "task1", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task2.1", TaskID: "task2", SlotsNeeded: 1, Group: groups[1]},
	}

	expectedToAllocate := []*MockTask{tasks[1]}
	expectedToRelease := []*MockTask{}

	logpattern.SetDisallowedNodesCacheTest(t, map[model.TaskID]*set.Set[string]{
		"task1": ptrs.Ptr(set.FromSlice([]string{"agent"})),
	})

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	toAllocate, toRelease := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}
