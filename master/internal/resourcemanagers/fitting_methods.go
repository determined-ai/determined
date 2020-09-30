package resourcemanagers

import "fmt"

// Hard Constraints

func slotsSatisfied(req *AllocateRequest, agent *agentState) bool {
	return req.SlotsNeeded <= agent.numEmptySlots()
}

func labelSatisfied(req *AllocateRequest, agent *agentState) bool {
	return req.Label == agent.label
}

// Soft Constraints

// BestFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is both most utilized and
// offers the fewest slots. This method should be used when the cluster is dominated by multi-slot
// applications.
func BestFit(_ *AllocateRequest, agent *agentState) float64 {
	return 1.0 / (1.0 + float64(agent.numEmptySlots()))
}

// WorstFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is least utilized. This
// method should be used when the cluster is dominated by single-slot applications.
func WorstFit(_ *AllocateRequest, agent *agentState) float64 {
	return float64(agent.numEmptySlots()) / float64(agent.numSlots())
}

// MakeFitFunction returns the corresponding fitting function.
func MakeFitFunction(fittingPolicy string) func(*AllocateRequest, *agentState) float64 {
	switch fittingPolicy {
	case "worst":
		return WorstFit
	case "best":
		return BestFit
	default:
		panic(fmt.Sprintf("invalid scheduler fit: %s", fittingPolicy))
	}
}
