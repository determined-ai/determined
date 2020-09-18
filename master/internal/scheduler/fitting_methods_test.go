package scheduler

import (
	"testing"

	cproto "github.com/determined-ai/determined/master/pkg/container"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func consumeSlots(agent *agentState, consume int) *agentState {
	req := &AllocateRequest{
		SlotsNeeded:  consume,
		CanTerminate: true,
	}
	container := newContainer(req, agent, req.SlotsNeeded, 0)
	agent.allocateFreeDevices(req.SlotsNeeded, cproto.ID(container.id))
	return agent
}

func TestBestFit(t *testing.T) {
	system := actor.NewSystem(t.Name())
	assert.Equal(t, BestFit(nil, consumeSlots(newMockAgent(t, system, "agent1", 1, ""), 1)), 1.0)
	assert.Equal(t, BestFit(nil, consumeSlots(newMockAgent(t, system, "agent2", 1, ""), 0)), 0.5)
	assert.Equal(t, BestFit(nil, consumeSlots(newMockAgent(t, system, "agent3", 9, ""), 0)), 0.1)
	assert.Equal(t, BestFit(nil, consumeSlots(newMockAgent(t, system, "agent4", 10, ""), 1)), 0.1)
}

func TestWorstFit(t *testing.T) {
	system := actor.NewSystem(t.Name())
	assert.Equal(t, WorstFit(nil, consumeSlots(newMockAgent(t, system, "agent1", 1, ""), 0)), 1.0)
	assert.Equal(t, WorstFit(nil, consumeSlots(newMockAgent(t, system, "agent2", 1, ""), 1)), 0.0)
	assert.Equal(t, WorstFit(nil, consumeSlots(newMockAgent(t, system, "agent3", 10, ""), 0)), 1.0)
	assert.Equal(t, WorstFit(nil, consumeSlots(newMockAgent(t, system, "agent4", 10, ""), 5)), 0.5)
}
