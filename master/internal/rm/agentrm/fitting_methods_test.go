package agentrm

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
)

func TestBestFit(t *testing.T) {
	assert.Equal(t, BestFit(
		&sproto.AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, "agent1", 0, 0, 0, 0),
	), 0.0)
	assert.Equal(t, BestFit(
		&sproto.AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, "agent2", 0, 0, 0, 1),
	), 0.0)
	assert.Equal(t, BestFit(
		&sproto.AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, "agent3", 0, 0, 2, 0),
	), 1.0/3.0)
	assert.Equal(t, BestFit(
		&sproto.AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, "agent4", 0, 0, 2, 1),
	), 0.5)
	assert.Equal(t, BestFit(
		&sproto.AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, "agent5", 1, 1, 100, 0),
	), 1.0)
	assert.Equal(t, BestFit(
		&sproto.AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, "agent6", 1, 0, 100, 0),
	), 0.5)
	assert.Equal(t, BestFit(
		&sproto.AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, "agent7", 9, 0, 100, 0),
	), 0.1)
	assert.Equal(t, BestFit(
		&sproto.AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, "agent8", 10, 1, 100, 0),
	), 0.1)
}

func TestWorstFit(t *testing.T) {
	assert.Equal(t, WorstFit(
		&sproto.AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, "agent1", 0, 0, 0, 0),
	), 0.0)
	assert.Equal(t, WorstFit(
		&sproto.AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, "agent2", 0, 0, 0, 1),
	), 0.0)
	assert.Equal(t, WorstFit(
		&sproto.AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, "agent3", 0, 0, 2, 0),
	), 1.0)
	assert.Equal(t, WorstFit(
		&sproto.AllocateRequest{SlotsNeeded: 0},
		newFakeAgentState(t, "agent4", 0, 0, 2, 1),
	), 0.5)
	assert.Equal(t, WorstFit(
		&sproto.AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, "agent5", 1, 0, 100, 0),
	), 1.0)
	assert.Equal(t, WorstFit(
		&sproto.AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, "agent6", 1, 1, 100, 0),
	), 0.0)
	assert.Equal(t, WorstFit(
		&sproto.AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, "agent7", 10, 0, 100, 0),
	), 1.0)
	assert.Equal(t, WorstFit(
		&sproto.AllocateRequest{SlotsNeeded: 1},
		newFakeAgentState(t, "agent8", 10, 5, 100, 0),
	), 0.5)
}
