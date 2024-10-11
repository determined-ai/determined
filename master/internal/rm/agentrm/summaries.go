package agentrm

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/device"
)

// newAgentSummary returns a new immutable view of the agent.
func newAgentSummary(state *agentState) sproto.AgentSummary {
	return sproto.AgentSummary{
		Name:   string(state.id),
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

func (rp *resourcePool) resourceSummaryFromAgentStates(
	agentInfo map[aproto.ID]*agentState,
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

	// If we have heterogenous slots, we default to`UNSPECIFIED` aka `device.ZeroSlot`.
	// We raise an error in the logs if there is more than one slot type.
	if len(deviceTypeCount) > 1 {
		rp.syslog.Errorf("resource pool has unspecified slot type with %v total slot types", len(deviceTypeCount))
	} else {
		for deviceType := range deviceTypeCount {
			summary.slotType = deviceType
		}
	}

	return summary
}
