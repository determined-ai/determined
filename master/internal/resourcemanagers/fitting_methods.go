package resourcemanagers

import (
	"fmt"
)

// Hard Constraints

func slotsSatisfied(req *AllocateRequest, agent *agentState) bool {
	return req.SlotsNeeded <= agent.numEmptySlots()
}

func labelSatisfied(req *AllocateRequest, agent *agentState) bool {
	return req.Label == agent.label
}

func maxZeroSlotContainersSatisfied(req *AllocateRequest, agent *agentState) bool {
	if req.SlotsNeeded == 0 {
		if agent.maxZeroSlotContainers == 0 {
			return false
		}
		return agent.numZeroSlotContainers() < agent.maxZeroSlotContainers
	}
	return true
}

func agentSlotUnusedSatisfied(_ *AllocateRequest, agent *agentState) bool {
	return agent.numUsedSlots() == 0
}

// Soft Constraints

// BestFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is both most utilized and
// offers the fewest slots. This method should be used when the cluster is dominated by multi-slot
// applications.
func BestFit(req *AllocateRequest, agent *agentState) float64 {
	switch {
	case agent.numUsedSlots() != 0 || req.SlotsNeeded != 0:
		return 1.0 / (1.0 + float64(agent.numEmptySlots()))
	case agent.maxZeroSlotContainers == 0:
		return 0.0
	default:
		return 1.0 / (1.0 + float64(agent.maxZeroSlotContainers-agent.numZeroSlotContainers()))
	}
}

// WorstFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is least utilized. This
// method should be used when the cluster is dominated by single-slot applications.
func WorstFit(req *AllocateRequest, agent *agentState) float64 {
	switch {
	case agent.numUsedSlots() != 0 || req.SlotsNeeded != 0:
		return float64(agent.numEmptySlots()) / float64(agent.numSlots())
	case agent.maxZeroSlotContainers == 0:
		return 0.0
	default:
		return float64(agent.maxZeroSlotContainers-agent.numZeroSlotContainers()) /
			float64(agent.maxZeroSlotContainers)
	}
}

// MakeFitFunction returns the corresponding fitting function.
func MakeFitFunction(fittingPolicy string) func(*AllocateRequest, *agentState) float64 {
	switch fittingPolicy {
	case worst:
		return WorstFit
	case best:
		return BestFit
	default:
		panic(fmt.Sprintf("invalid scheduler fit: %s", fittingPolicy))
	}
}
