package rm

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/sproto"
)

// Hard Constraints

func slotsSatisfied(req *sproto.AllocateRequest, agent *AgentState) bool {
	return req.SlotsNeeded <= agent.NumEmptySlots()
}

func labelSatisfied(req *sproto.AllocateRequest, agent *AgentState) bool {
	return req.AgentLabel == agent.Label
}

func maxZeroSlotContainersSatisfied(req *sproto.AllocateRequest, agent *AgentState) bool {
	if req.SlotsNeeded == 0 {
		return agent.NumEmptyZeroSlots() > 0
	}
	return true
}

func agentSlotUnusedSatisfied(_ *sproto.AllocateRequest, agent *AgentState) bool {
	return agent.NumUsedSlots() == 0
}

// Soft Constraints

// BestFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is both most utilized and
// offers the fewest slots. This method should be used when the cluster is dominated by multi-slot
// applications.
func BestFit(req *sproto.AllocateRequest, agent *AgentState) float64 {
	switch {
	case agent.NumUsedSlots() != 0 || req.SlotsNeeded != 0:
		return 1.0 / (1.0 + float64(agent.NumEmptySlots()))
	case agent.NumZeroSlots() == 0:
		return 0.0
	default:
		return 1.0 / (1.0 + float64(agent.NumEmptyZeroSlots()))
	}
}

// WorstFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is least utilized. This
// method should be used when the cluster is dominated by single-slot applications.
func WorstFit(req *sproto.AllocateRequest, agent *AgentState) float64 {
	switch {
	case agent.NumUsedSlots() != 0 || req.SlotsNeeded != 0:
		return float64(agent.NumEmptySlots()) / float64(agent.NumSlots())
	case agent.NumZeroSlots() == 0:
		return 0.0
	default:
		return float64(agent.NumEmptyZeroSlots()) / float64(agent.NumZeroSlots())
	}
}

// MakeFitFunction returns the corresponding fitting function.
func MakeFitFunction(fittingPolicy string) func(
	*sproto.AllocateRequest, *AgentState) float64 {
	switch fittingPolicy {
	case worst:
		return WorstFit
	case best:
		return BestFit
	default:
		panic(fmt.Sprintf("invalid scheduler fit: %s", fittingPolicy))
	}
}
