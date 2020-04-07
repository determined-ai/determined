package scheduler

// Hard Constraints

func slotsSatisfied(task *Task, agent *agentState) bool {
	return task.SlotsNeeded() <= agent.numEmptySlots()
}

func labelSatisfied(task *Task, agent *agentState) bool {
	return task.agentLabel == agent.label
}

// Soft Constraints

// BestFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is both most utilized and
// offers the fewest slots. This method should be used when the cluster is dominated by multi-slot
// applications.
func BestFit(_ *Task, agent *agentState) float64 {
	return 1.0 / (1.0 + float64(agent.numEmptySlots()))
}

// WorstFit returns a float affinity score between 0 and 1 for the affinity between the task and
// the agent. This method attempts to allocate tasks to the agent that is least utilized. This
// method should be used when the cluster is dominated by single-slot applications.
func WorstFit(_ *Task, agent *agentState) float64 {
	return float64(agent.numEmptySlots()) / float64(agent.numSlots())
}
