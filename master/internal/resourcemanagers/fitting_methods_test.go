package resourcemanagers

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestBestFit(t *testing.T) {
	system := actor.NewSystem(t.Name())
	assert.Equal(t, BestFit(
		&AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, system, "agent1", "", 0, 0, 0, 0),
	), 0.0)
	assert.Equal(t, BestFit(
		&AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, system, "agent2", "", 0, 0, 0, 1),
	), 0.0)
	assert.Equal(t, BestFit(
		&AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, system, "agent3", "", 0, 0, 2, 0),
	), 1.0/3.0)
	assert.Equal(t, BestFit(
		&AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, system, "agent4", "", 0, 0, 2, 1),
	), 0.5)
	assert.Equal(t, BestFit(
		&AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, system, "agent5", "", 1, 1, 100, 0),
	), 1.0)
	assert.Equal(t, BestFit(
		&AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, system, "agent6", "", 1, 0, 100, 0),
	), 0.5)
	assert.Equal(t, BestFit(
		&AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, system, "agent7", "", 9, 0, 100, 0),
	), 0.1)
	assert.Equal(t, BestFit(
		&AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, system, "agent8", "", 10, 1, 100, 0),
	), 0.1)
}

func TestWorstFit(t *testing.T) {
	system := actor.NewSystem(t.Name())
	assert.Equal(t, WorstFit(
		&AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, system, "agent1", "", 0, 0, 0, 0),
	), 0.0)
	assert.Equal(t, WorstFit(
		&AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, system, "agent2", "", 0, 0, 0, 1),
	), 0.0)
	assert.Equal(t, WorstFit(
		&AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, system, "agent3", "", 0, 0, 2, 0),
	), 1.0)
	assert.Equal(t, WorstFit(
		&AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, system, "agent4", "", 0, 0, 2, 1),
	), 0.5)
	assert.Equal(t, WorstFit(
		&AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, system, "agent5", "", 1, 0, 100, 0),
	), 1.0)
	assert.Equal(t, WorstFit(
		&AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, system, "agent6", "", 1, 1, 100, 0),
	), 0.0)
	assert.Equal(t, WorstFit(
		&AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, system, "agent7", "", 10, 0, 100, 0),
	), 1.0)
	assert.Equal(t, WorstFit(
		&AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, system, "agent8", "", 10, 5, 100, 0),
	), 0.5)
}
