package scheduler

import "github.com/determined-ai/determined/master/pkg/actor"

// FilterableView keeps track of tasks and agents that pass the task and agent filters.
// The `TaskSummary`s and `AgentSummary` should not be modified because a reference to
// this struct is contained in another goroutine.
type FilterableView struct {
	tasks           map[RequestID]*TaskSummary
	filteredAgents  map[*actor.Ref]*AgentSummary
	connectedAgents map[*actor.Ref]*AgentSummary
	taskFilter      func(*AssignRequest, *ResourceAssigned) bool
	agentFilter     func(*agentState) bool
}

// Return a view of the scheduler state that is relevant to the provisioner. Specifically, the
// provisioner cares about (1) idle agents (2) pending tasks.
func newProvisionerView(provisionerSlotsPerInstance int) *FilterableView {
	return &FilterableView{
		tasks:           make(map[RequestID]*TaskSummary),
		filteredAgents:  make(map[*actor.Ref]*AgentSummary),
		connectedAgents: make(map[*actor.Ref]*AgentSummary),
		taskFilter:      schedulableTaskFilter(provisionerSlotsPerInstance),
		agentFilter:     idleAgentFilter,
	}
}

func schedulableTaskFilter(
	provisionerSlotsPerInstance int,
) func(*AssignRequest, *ResourceAssigned) bool {
	// We only tell the provisioner about pending tasks that are compatible with the
	// provisioner's configured instance type.
	return func(req *AssignRequest, assigned *ResourceAssigned) bool {
		slotsNeeded := req.SlotsNeeded

		switch {
		case assigned != nil && len(assigned.Assignments) > 0:
			return false
		// TODO(DET-4035): This code is duplicated from the fitting functions in the
		// scheduler. To determine is a task is schedulable, we would ideally interface
		// with the scheduler in some way and not duplicate this logic.
		case slotsNeeded <= provisionerSlotsPerInstance:
			return true
		case slotsNeeded%provisionerSlotsPerInstance == 0:
			return true
		default:
			return false
		}
	}
}

func idleAgentFilter(agent *agentState) bool {
	return agent.numUsedSlots() == 0 && len(agent.zeroSlotContainers) == 0
}

// Update updates the FilterableView with the current state of the cluster.
func (v *FilterableView) Update(rp *DefaultRP) (ViewSnapshot, bool) {
	// We must evaluate v.updateTasks(cluster) and v.updateAgents(cluster)
	// before taking the logical or of the results to ensure that short circuit
	// evaluation of booleans expressions don't prevent the updating of agents.
	tasksUpdateMade := v.updateTasks(rp)
	agentsUpdateMade := v.updateAgents(rp)
	return v.newSnapshot(), tasksUpdateMade || agentsUpdateMade
}

func (v *FilterableView) updateTasks(rp *DefaultRP) bool {
	newTasks := make(map[RequestID]*TaskSummary)

	for iterator := rp.reqList.iterator(); iterator.next(); {
		req := iterator.value()

		if v.taskFilter(req, rp.reqList.GetAssignments(req.Handler)) {
			newTasks[req.ID] = getTaskSummary(rp.reqList, req.ID)
		}
	}

	updateMade := false
	if len(newTasks) != len(v.tasks) {
		updateMade = true
	} else {
		for _, newTask := range newTasks {
			oldTask, ok := v.tasks[newTask.ID]
			if !ok || !oldTask.equals(newTask) {
				updateMade = true
			}
		}
	}

	v.tasks = newTasks
	return updateMade
}

func (v *FilterableView) updateAgents(rp *DefaultRP) bool {
	newFilteredAgents := make(map[*actor.Ref]*AgentSummary)
	newConnectedAgents := make(map[*actor.Ref]*AgentSummary)

	for actorRef, state := range rp.agents {
		agentSummary := newAgentSummary(state)
		if v.agentFilter(state) {
			newFilteredAgents[actorRef] = &agentSummary
		}
		newConnectedAgents[actorRef] = &agentSummary
	}

	haveAgentsUpdated := func(updatedAgents, previousAgents map[*actor.Ref]*AgentSummary) bool {
		updateMade := false
		if len(updatedAgents) != len(previousAgents) {
			updateMade = true
		} else {
			for agentRef, newAgent := range updatedAgents {
				oldAgent, ok := previousAgents[agentRef]
				if !ok || !oldAgent.equals(newAgent) {
					updateMade = true
				}
			}
		}
		return updateMade
	}

	agentsUpdated := haveAgentsUpdated(newFilteredAgents, v.filteredAgents) || haveAgentsUpdated(
		newConnectedAgents, v.connectedAgents)

	v.filteredAgents = newFilteredAgents
	v.connectedAgents = newConnectedAgents
	return agentsUpdated
}

func (v *FilterableView) newSnapshot() ViewSnapshot {
	tasks := make([]*TaskSummary, 0, len(v.tasks))
	for _, taskSummary := range v.tasks {
		tasks = append(tasks, taskSummary)
	}
	connectedAgents := make([]*AgentSummary, 0, len(v.connectedAgents))
	for _, agent := range v.connectedAgents {
		connectedAgents = append(connectedAgents, agent)
	}
	filteredAgents := make([]*AgentSummary, 0, len(v.filteredAgents))
	for _, agent := range v.filteredAgents {
		filteredAgents = append(filteredAgents, agent)
	}
	return ViewSnapshot{Tasks: tasks, ConnectedAgents: connectedAgents, IdleAgents: filteredAgents}
}
