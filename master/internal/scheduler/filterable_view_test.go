package scheduler

import (
	"fmt"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func (c *Cluster) addTask(inTask *Task) *Task {
	task := newTask(inTask)

	c.tasksByID[task.ID] = task
	c.tasksByHandler[task.handler] = task
	c.taskList.Add(task)

	return task
}

func (c *Cluster) addAgent(
	t *testing.T,
	system *actor.System,
	agentID string,
	numSlots int,
	numUsedSlots int,
	numZeroSlotContainers int,
) {
	agent := createAgent(t, system, agentID, numSlots, numUsedSlots, numZeroSlotContainers)
	c.agents[agent.handler] = agent
}

func createAgents(
	t *testing.T,
	system *actor.System,
	agentIDPrefix string,
	numIdleAgents int,
	numActiveAgents int,
) []*agentState {
	agents := make([]*agentState, 0, numIdleAgents+numActiveAgents)
	for i := 0; i < numIdleAgents; i++ {
		agentID := fmt.Sprintf("%s-%d", agentIDPrefix, len(agents))
		agents = append(agents, createAgent(t, system, agentID+"-c", 4, 0, 0))
	}
	for i := 0; i < numActiveAgents; i++ {
		agentID := fmt.Sprintf("%s-%d", agentIDPrefix, len(agents))
		agents = append(agents, createAgent(t, system, agentID+"-c", 4, 1, 0))
	}
	return agents
}

func createAgent(
	t *testing.T,
	system *actor.System,
	agentID string,
	numSlots int,
	numUsedSlots int,
	numZeroSlotContainers int,
) *agentState {
	state := newMockAgent(t, system, agentID, numSlots, "")
	state.containers = createContainers(agentID+"-c", numUsedSlots, numZeroSlotContainers)
	return state
}

func createContainers(
	idPrefix string,
	numSlots int,
	numZeroSlotContainers int,
) map[ContainerID]*container {
	containers := make(map[ContainerID]*container)
	for i := 0; i < numSlots; i++ {
		containers[ContainerID(fmt.Sprintf("%s-%d", idPrefix, i))] = &container{slots: 1}
	}
	for i := 0; i < numZeroSlotContainers; i++ {
		containers[ContainerID(fmt.Sprintf("%s-%d", idPrefix, i))] = &container{slots: 0}
	}
	return containers
}

func (snapshot1 *ViewSnapshot) isSubset(snapshot2 *ViewSnapshot) bool {
	tasksSub := tasksIsSubset(snapshot1.Tasks, snapshot2.Tasks)
	agentsSub := agentsIsSubset(snapshot1.Agents, snapshot2.Agents)
	return tasksSub && agentsSub
}

func (snapshot1 *ViewSnapshot) difference(snapshot2 *ViewSnapshot) *ViewSnapshot {
	return &ViewSnapshot{
		Tasks:  tasksDifference(snapshot1.Tasks, snapshot2.Tasks),
		Agents: agentsDifference(snapshot1.Agents, snapshot2.Agents),
	}
}

func areEqual(snapshot1 *ViewSnapshot, snapshot2 *ViewSnapshot) bool {
	if len(snapshot1.Tasks) != len(snapshot2.Tasks) ||
		len(snapshot1.Agents) != len(snapshot2.Tasks) {
		return false
	}
	tasksDiff := tasksDifference(snapshot1.Tasks, snapshot2.Tasks)
	agentsDiff := agentsDifference(snapshot1.Agents, snapshot2.Agents)
	return len(tasksDiff) == 0 && len(agentsDiff) == 0
}

func taskIsMember(tasks []*TaskSummary, task *TaskSummary) bool {
	for _, candidate := range tasks {
		if candidate.equals(task) {
			return true
		}
	}

	return false
}

func tasksIsSubset(tasks1 []*TaskSummary, tasks2 []*TaskSummary) bool {
	for _, task := range tasks1 {
		if !taskIsMember(tasks2, task) {
			return false
		}
	}

	return true
}

func tasksDifference(tasks1 []*TaskSummary, tasks2 []*TaskSummary) []*TaskSummary {
	var difference []*TaskSummary
	for _, task := range tasks2 {
		if !taskIsMember(tasks1, task) {
			difference = append(difference, task)
		}
	}

	return difference
}

func agentIsMember(agents []*AgentSummary, agent *AgentSummary) bool {
	for _, candidate := range agents {
		if candidate.equals(agent) {
			return true
		}
	}

	return false
}

func agentsIsSubset(agents1 []*AgentSummary, agents2 []*AgentSummary) bool {
	for _, agent := range agents1 {
		if !agentIsMember(agents2, agent) {
			return false
		}
	}

	return true
}

func agentsDifference(agents1 []*AgentSummary, agents2 []*AgentSummary) []*AgentSummary {
	var difference []*AgentSummary
	for _, agent := range agents2 {
		if !agentIsMember(agents1, agent) {
			difference = append(difference, agent)
		}
	}

	return difference
}

func addTask(
	t *testing.T,
	system *actor.System,
	c *Cluster,
	taskID string,
	state taskState,
	slotsNeeded int,
) *Task {
	task := c.addTask(&Task{
		ID:           TaskID(taskID),
		group:        c.getOrCreateGroup(newGroup(t, system, taskID+"-group"), nil),
		handler:      newGroup(t, system, taskID+"-handler"),
		slotsNeeded:  slotsNeeded,
		canTerminate: true,
		state:        taskPending,
	})
	task.state = state
	return task
}

func TestBasic(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	agents = append(agents, createAgent(t, system, "agentx", 1, 0, 1))
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	addTask(t, system, c, "task1", taskRunning, 1)
	addTask(t, system, c, "task2", taskPending, 1)
	addTask(t, system, c, "task3", taskPending, 5)

	snapshot1, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot1.Agents))
	assert.Equal(t, 1, len(snapshot1.Tasks))
	assert.Check(t, updated)
}

func TestNoUpdate(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	addTask(t, system, c, "task1", taskRunning, 1)
	addTask(t, system, c, "task2", taskPending, 1)

	snapshot1, _ := c.provisionerView.Update(c)
	addTask(t, system, c, "task3", taskRunning, 1)
	addTask(t, system, c, "task4", taskTerminated, 1)
	addTask(t, system, c, "task5", taskTerminating, 1)
	c.addAgent(t, system, "agent-a", 4, 1, 0)

	snapshot2, updated := c.provisionerView.Update(c)
	assert.Check(t, !updated)
	assert.Check(t, areEqual(&snapshot1, &snapshot2))
}

func TestAddTask(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	addTask(t, system, c, "task1", taskRunning, 1)
	addTask(t, system, c, "task2", taskPending, 1)

	snapshot1, _ := c.provisionerView.Update(c)
	addTask(t, system, c, "task3", taskPending, 1)
	addTask(t, system, c, "task4", taskRunning, 1)
	addTask(t, system, c, "task5", taskTerminated, 1)
	addTask(t, system, c, "task6", taskTerminating, 1)
	snapshot2, updated := c.provisionerView.Update(c)

	isSubset := snapshot1.isSubset(&snapshot2)
	difference := snapshot1.difference(&snapshot2)
	assert.Check(t, updated)
	assert.Check(t, isSubset)
	assert.Equal(t, 0, len(difference.Agents))
	assert.Equal(t, 1, len(difference.Tasks))
}

func TestAddIdleAgent(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	addTask(t, system, c, "task1", taskRunning, 1)
	addTask(t, system, c, "task2", taskPending, 1)

	snapshot1, _ := c.provisionerView.Update(c)
	c.addAgent(t, system, "agent-a1", 4, 0, 0)
	c.addAgent(t, system, "agent-a2", 4, 1, 0)
	snapshot2, updated := c.provisionerView.Update(c)

	isSubset := snapshot1.isSubset(&snapshot2)
	difference := snapshot1.difference(&snapshot2)
	assert.Check(t, updated)
	assert.Check(t, isSubset)
	assert.Equal(t, 0, len(difference.Tasks))
	assert.Equal(t, 1, len(difference.Agents))
}

func TestRemoveAgent(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)

	snapshot1, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot1.Agents))
	assert.Equal(t, 0, len(snapshot1.Tasks))
	assert.Check(t, updated)

	delete(c.agents, agents[0].handler)

	snapshot2, updated := c.provisionerView.Update(c)

	assert.Equal(t, 0, len(snapshot2.Agents))
	assert.Equal(t, 0, len(snapshot2.Tasks))
	assert.Check(t, updated)
}

func TestTaskStateChange(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	pendingTask := addTask(t, system, c, "task1", taskPending, 1)

	snapshot1, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot1.Agents))
	assert.Equal(t, 1, len(snapshot1.Tasks))
	assert.Check(t, updated)

	pendingTask.mustTransition(taskRunning)

	snapshot2, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot2.Agents))
	assert.Equal(t, 0, len(snapshot2.Tasks))
	assert.Check(t, updated)
}

func TestTaskSlotsNeededChange(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	pendingTask := addTask(t, system, c, "task1", taskPending, 1)

	snapshot1, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot1.Agents))
	assert.Equal(t, 1, len(snapshot1.Tasks))
	assert.Check(t, updated)

	pendingTask.slotsNeeded = 4

	snapshot2, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot2.Agents))
	assert.Equal(t, 1, len(snapshot2.Tasks))
	assert.Check(t, updated)
}

func TestAgentStateChange(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)

	snapshot1, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot1.Agents))
	assert.Equal(t, 0, len(snapshot1.Tasks))
	assert.Check(t, updated)

	agents[0].containers = createContainers("agent-c", 1, 0)

	snapshot2, updated := c.provisionerView.Update(c)

	assert.Equal(t, 0, len(snapshot2.Agents))
	assert.Equal(t, 0, len(snapshot2.Tasks))
	assert.Check(t, updated)
}
