//go:build integration
// +build integration

package agentrm

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/set"
)

func TestFindFitDisallowedNodes(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{
		newFakeAgentState(t, system, "agent1", 4, 0, 100, 0),
		newFakeAgentState(t, system, "agent2", 4, 0, 100, 0),
	}
	agentsByHandler, _ := byHandler(agents...)

	logpattern.SetDisallowedNodesCacheTest(t, map[model.TaskID]*set.Set[string]{
		"noAgents":    ptrs.Ptr(set.FromSlice([]string{"agent1", "agent2"})),
		"notOnAgent1": ptrs.Ptr(set.FromSlice([]string{"agent1"})),
		"notOnAgent2": ptrs.Ptr(set.FromSlice([]string{"agent2"})),
	})

	task := &sproto.AllocateRequest{
		AllocationID: "a",
		SlotsNeeded:  1,
		TaskID:       "noAgents",
	}
	fits := findFits(task, agentsByHandler, BestFit, false)
	assert.Assert(t, len(fits) == 0)

	task = &sproto.AllocateRequest{
		AllocationID: "a",
		SlotsNeeded:  1,
		TaskID:       "notOnAgent1",
	}
	fits = findFits(task, agentsByHandler, BestFit, false)
	assert.Assert(t, len(fits) == 1)
	assert.Equal(t, fits[0].Agent, agents[1])

	task = &sproto.AllocateRequest{
		AllocationID: "a",
		SlotsNeeded:  1,
		TaskID:       "notOnAgent2",
	}
	fits = findFits(task, agentsByHandler, BestFit, false)
	assert.Assert(t, len(fits) == 1)
	assert.Equal(t, fits[0].Agent, agents[0])
}
