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

func newProvisionerView(provisionerSlotsPerInstance int) *FilterableView {
	return &FilterableView{
		tasks:       make(map[TaskID]*TaskSummary),
		agents:      make(map[*actor.Ref]*AgentSummary),
		taskFilter:  schedulableTaskFilter(provisionerSlotsPerInstance),
		agentFilter: idleAgentFilter,
	}
}

func schedulableTaskFilter(provisionerSlotsPerInstance int) func(*Task) bool {
	return func(task *Task) bool {
		pending := task.state == taskPending
		zeroOrSingleSlotTask := task.SlotsNeeded() == 0 || task.SlotsNeeded() == 1
		multiSlotTaskFits := task.SlotsNeeded()%provisionerSlotsPerInstance == 0
		return pending && (zeroOrSingleSlotTask || multiSlotTaskFits)
	}
}

func idleAgentFilter(agent *agentState) bool {
	return len(agent.containers) == 0
}

// Update updates the FilterableView with the current state of the cluster.
func (v *FilterableView) Update(cluster *Cluster) (ViewSnapshot, bool) {
	// We must evaluate v.updateTasks(cluster) and v.updateAgents(cluster)
	// before taking the logical or of the results to ensure that short circuit
	// evaluation of booleans expressions don't prevent the updating of agents.
	tasksUpdateMade := v.updateTasks(cluster)
	agentsUpdateMade := v.updateAgents(cluster)
	return v.newSnapshot(), tasksUpdateMade || agentsUpdateMade
}

func (v *FilterableView) updateTasks(cluster *Cluster) bool {
	updateMade := false
	tasks := make(map[TaskID]*TaskSummary)
	for iterator := cluster.taskList.iterator(); iterator.next(); {
		task := iterator.value()

		if v.taskFilter(task) {
			taskSummary := newTaskSummary(task)
			if summary, ok := v.tasks[task.ID]; !ok || !summary.equals(&taskSummary) {
				tasks[task.ID] = &taskSummary
				updateMade = true
			} else {
				tasks[task.ID] = summary
			}
		} else if _, ok := v.tasks[task.ID]; ok {
			// Indicate that we've made an update since the new map
			// does not have the task anymore.
			updateMade = true
		}
	}

	// The case of `updateMade` is false but `len(tasks) != len(v.tasks)`
	// is when a task was deleted from the map but no other relevant changes were made.
	updateMade = updateMade || len(tasks) != len(v.tasks)
	v.tasks = tasks
	return updateMade
}

func (v *FilterableView) updateAgents(cluster *Cluster) bool {
	updateMade := false
	agents := make(map[*actor.Ref]*AgentSummary)
	for actorRef, state := range cluster.agents {
		if v.agentFilter(state) {
			agentSummary := newAgentSummary(state)
			if summary, ok := v.agents[actorRef]; !ok || *summary != agentSummary {
				agents[actorRef] = &agentSummary
				updateMade = true
			} else {
				agents[actorRef] = summary
			}
		} else if _, ok := v.agents[actorRef]; ok {
			// Indicate that we've made an update since the new map
			// does not have the agent anymore.
			updateMade = true
		}
	}

	// The case of `updateMade` is false but `len(agents) != len(v.agents)`
	// is when an agent was deleted from the map but no other relevant changes were made.
	updateMade = updateMade || len(agents) != len(v.agents)
	v.agents = agents
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
