package scheduler

import "github.com/determined-ai/determined/master/pkg/actor"

// FilterableView keeps track of tasks and agents that pass the task and agent filters.
// The `TaskSummary`s and `AgentSummary` should not be modified because a reference to
// this struct is contained in another goroutine.
type FilterableView struct {
	tasks       map[TaskID]*TaskSummary
	agents      map[*actor.Ref]*AgentSummary
	taskFilter  func(*Task) bool
	agentFilter func(*agentState) bool
}

// Return a view of the scheduler state that is relevant to the provisioner. Specifically, the
// provisioner cares about (1) idle agents (2) pending tasks.
func newProvisionerView(provisionerSlotsPerInstance int) *FilterableView {
	return &FilterableView{
		tasks:       make(map[TaskID]*TaskSummary),
		agents:      make(map[*actor.Ref]*AgentSummary),
		taskFilter:  schedulableTaskFilter(provisionerSlotsPerInstance),
		agentFilter: idleAgentFilter,
	}
}

func schedulableTaskFilter(provisionerSlotsPerInstance int) func(*Task) bool {
	// We only tell the provisioner about pending tasks that are compatible with the
	// provisioner's configured instance type.
	return func(task *Task) bool {
		slotsNeeded := task.SlotsNeeded()

		switch {
		case task.state != taskPending:
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
	return len(agent.containers) == 0
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
	newTasks := make(map[TaskID]*TaskSummary)

	for iterator := rp.taskList.iterator(); iterator.next(); {
		task := iterator.value()

		if v.taskFilter(task) {
			taskSummary := newTaskSummary(task)
			newTasks[task.ID] = &taskSummary
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
	newAgents := make(map[*actor.Ref]*AgentSummary)

	for actorRef, state := range rp.agents {
		if v.agentFilter(state) {
			agentSummary := newAgentSummary(state)
			newAgents[actorRef] = &agentSummary
		}
	}

	updateMade := false
	if len(newAgents) != len(v.agents) {
		updateMade = true
	} else {
		for agentRef, newAgent := range newAgents {
			oldAgent, ok := v.agents[agentRef]
			if !ok || !oldAgent.equals(newAgent) {
				updateMade = true
			}
		}
	}

	v.agents = newAgents
	return updateMade
}

func (v *FilterableView) newSnapshot() ViewSnapshot {
	tasks := make([]*TaskSummary, 0, len(v.tasks))
	for _, taskSummary := range v.tasks {
		tasks = append(tasks, taskSummary)
	}
	agents := make([]*AgentSummary, 0, len(v.agents))
	for _, agent := range v.agents {
		agents = append(agents, agent)
	}
	return ViewSnapshot{Tasks: tasks, Agents: agents}
}
