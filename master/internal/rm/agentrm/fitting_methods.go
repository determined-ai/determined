package agentrm

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/sproto"
)

// Hard Constraints

func slotsSatisfied(req *sproto.AllocateRequest, agent *agentState) bool {
	return req.SlotsNeeded <= agent.numEmptySlots()
}

func maxZeroSlotContainersSatisfied(req *sproto.AllocateRequest, agent *agentState) bool {
	if req.SlotsNeeded == 0 {
		return agent.numEmptyZeroSlots() > 0
	}
	return true
}

func agentSlotUnusedSatisfied(_ *sproto.AllocateRequest, agent *agentState) bool {
	return agent.numUsedSlots() == 0
}

// Soft Constraints

// BestFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is both most utilized and
// offers the fewest slots. This method should be used when the cluster is dominated by multi-slot
// applications.
func BestFit(req *sproto.AllocateRequest, agent *agentState) float64 {
	switch {
	case agent.numUsedSlots() != 0 || req.SlotsNeeded != 0:
		return 1.0 / (1.0 + float64(agent.numEmptySlots()))
	case agent.numZeroSlots() == 0:
		return 0.0
	default:
		return 1.0 / (1.0 + float64(agent.numEmptyZeroSlots()))
	}
}

// WorstFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is least utilized. This
// method should be used when the cluster is dominated by single-slot applications.
func WorstFit(req *sproto.AllocateRequest, agent *agentState) float64 {
	switch {
	case agent.numUsedSlots() != 0 || req.SlotsNeeded != 0:
		return float64(agent.numEmptySlots()) / float64(agent.numSlots())
	case agent.numZeroSlots() == 0:
		return 0.0
	default:
		return float64(agent.numEmptyZeroSlots()) / float64(agent.numZeroSlots())
	}
}

// MakeFitFunction returns the corresponding fitting function.
func MakeFitFunction(fittingPolicy string) func(
	*sproto.AllocateRequest, *agentState) float64 {
	switch fittingPolicy {
	case worst:
		return WorstFit
	case best:
		return BestFit
	default:
		panic(fmt.Sprintf("invalid scheduler fit: %s", fittingPolicy))
	}
}
