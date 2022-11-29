package agentrm

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
)

// newAgentSummary returns a new immutable view of the agent.
func newAgentSummary(state *agentState) sproto.AgentSummary {
	return sproto.AgentSummary{
		Name:   state.Handler.Address().Local(),
		IsIdle: state.idle(),
	}
}

// resourceSummary is a summary of the resource available/used by a resource pool.
type resourceSummary struct {
	numAgents              int
	numTotalSlots          int
	numActiveSlots         int
	maxNumAuxContainers    int
	numActiveAuxContainers int
	slotType               device.Type
}

func resourceSummaryFromAgentStates(
	agentInfo map[*actor.Ref]*agentState,
) resourceSummary {
	summary := resourceSummary{
		numTotalSlots:          0,
		numActiveSlots:         0,
		maxNumAuxContainers:    0,
		numActiveAuxContainers: 0,
		slotType:               device.ZeroSlot,
	}

	deviceTypeCount := make(map[device.Type]int)

	for _, agentState := range agentInfo {
		summary.numAgents++
		summary.numTotalSlots += agentState.numSlots()
		summary.numActiveSlots += agentState.numUsedSlots()
		summary.maxNumAuxContainers += agentState.numZeroSlots()
		summary.numActiveAuxContainers += agentState.numUsedZeroSlots()
		for agentDevice := range agentState.Devices {
			deviceTypeCount[agentDevice.Type]++
		}
	}

	// If we have homogenous slots, get their type. Otherwise, we default to
	// `UNSPECIFIED` aka `device.ZeroSlot`, although it may be an error/warning.
	if len(deviceTypeCount) == 1 {
		for deviceType := range deviceTypeCount {
			summary.slotType = deviceType
		}
	}

	return summary
}
