package scheduler

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	cproto "github.com/determined-ai/determined/master/pkg/container"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func (d *DefaultRP) addRequest(req *AssignRequest, assigned *ResourceAssigned) {
	d.reqList.Add(req)
	d.reqList.SetAssignments(req.Handler, assigned)
}

func (d *DefaultRP) addAgent(
	t *testing.T,
	system *actor.System,
	agentID string,
	numSlots int,
	numUsedSlots int,
	numZeroSlotContainers int,
) {
	agent := createAgent(t, system, agentID, numSlots, numUsedSlots, numZeroSlotContainers)
	d.agents[agent.handler] = agent
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
	i := 0
	for ix := range state.devices {
		if i < numUsedSlots {
			id := cproto.ID(uuid.New().String())
			state.devices[ix] = &id
		}
	}
	for i := 0; i < numZeroSlotContainers; i++ {
		state.zeroSlotContainers[cproto.ID(uuid.New().String())] = true
	}
	return state
}

func (snapshot1 *ViewSnapshot) isSubset(snapshot2 *ViewSnapshot) bool {
	tasksSub := tasksIsSubset(snapshot1.Tasks, snapshot2.Tasks)
	idleAgentsSub := agentsIsSubset(snapshot1.IdleAgents, snapshot2.IdleAgents)
	connectedAgentsSub := agentsIsSubset(snapshot1.ConnectedAgents, snapshot2.ConnectedAgents)
	return tasksSub && idleAgentsSub && connectedAgentsSub
}

func (snapshot1 *ViewSnapshot) difference(snapshot2 *ViewSnapshot) *ViewSnapshot {
	return &ViewSnapshot{
		Tasks:           tasksDifference(snapshot1.Tasks, snapshot2.Tasks),
		IdleAgents:      agentsDifference(snapshot1.IdleAgents, snapshot2.IdleAgents),
		ConnectedAgents: agentsDifference(snapshot1.ConnectedAgents, snapshot2.ConnectedAgents),
	}
}

func areEqual(snapshot1 *ViewSnapshot, snapshot2 *ViewSnapshot) bool {
	if len(snapshot1.Tasks) != len(snapshot2.Tasks) ||
		len(snapshot1.IdleAgents) != len(snapshot2.Tasks) {
		return false
	}
	tasksDiff := tasksDifference(snapshot1.Tasks, snapshot2.Tasks)
	idleAgentsDiff := agentsDifference(snapshot1.IdleAgents, snapshot2.IdleAgents)
	connectedAgentsDiff := agentsDifference(snapshot1.ConnectedAgents, snapshot2.ConnectedAgents)
	return len(tasksDiff) == 0 && len(idleAgentsDiff) == 0 && len(connectedAgentsDiff) == 0
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
	d *DefaultRP,
	taskID string,
	numAssigned int,
	slotsNeeded int,
) *AssignRequest {
	req := &AssignRequest{
		ID:           RequestID(taskID),
		Group:        newGroup(t, system, taskID+"-group"),
		Handler:      newGroup(t, system, taskID+"-handler"),
		SlotsNeeded:  slotsNeeded,
		CanTerminate: true,
	}
	assigned := &ResourceAssigned{Assignments: []Assignment{}}
	for i := 0; i < numAssigned; i++ {
		assigned.Assignments = append(assigned.Assignments, containerAssignment{})
	}
	d.addRequest(req, assigned)
	return req
}

func TestBasic(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	agents = append(agents, createAgent(t, system, "agentx", 1, 0, 1))
	var tasks []*actor.Ref
	d := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	d.provisionerView = newProvisionerView(4)
	addTask(t, system, d, "task1", 1, 1)
	addTask(t, system, d, "task2", 0, 1)
	addTask(t, system, d, "task3", 0, 5)

	snapshot1, updated := d.provisionerView.Update(d)

	assert.Equal(t, 1, len(snapshot1.IdleAgents))
	assert.Equal(t, 1, len(snapshot1.Tasks))
	assert.Equal(t, 7, len(snapshot1.ConnectedAgents))
	assert.Check(t, updated)
}

func TestNoUpdate(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	addTask(t, system, c, "task1", 1, 1)
	addTask(t, system, c, "task2", 0, 1)

	snapshot1, _ := c.provisionerView.Update(c)
	addTask(t, system, c, "task3", 1, 1)
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
	addTask(t, system, c, "task1", 1, 1)
	addTask(t, system, c, "task2", 0, 1)

	snapshot1, _ := c.provisionerView.Update(c)
	addTask(t, system, c, "task3", 0, 1)
	addTask(t, system, c, "task4", 1, 1)
	snapshot2, updated := c.provisionerView.Update(c)

	isSubset := snapshot1.isSubset(&snapshot2)
	difference := snapshot1.difference(&snapshot2)
	assert.Check(t, updated)
	assert.Check(t, isSubset)
	assert.Equal(t, 0, len(difference.IdleAgents))
	assert.Equal(t, 1, len(difference.Tasks))
	assert.Equal(t, 0, len(difference.ConnectedAgents))
}

func TestAddIdleAgent(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	addTask(t, system, c, "task1", 1, 1)
	addTask(t, system, c, "task2", 0, 1)

	snapshot1, _ := c.provisionerView.Update(c)
	c.addAgent(t, system, "agent-a1", 4, 0, 0)
	c.addAgent(t, system, "agent-a2", 4, 1, 0)
	snapshot2, updated := c.provisionerView.Update(c)

	isSubset := snapshot1.isSubset(&snapshot2)
	difference := snapshot1.difference(&snapshot2)
	assert.Check(t, updated)
	assert.Check(t, isSubset)
	assert.Equal(t, 0, len(difference.Tasks))
	assert.Equal(t, 1, len(difference.IdleAgents))
	assert.Equal(t, 2, len(difference.ConnectedAgents))
}

func TestRemoveAgent(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)

	snapshot1, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot1.IdleAgents))
	assert.Equal(t, 0, len(snapshot1.Tasks))
	assert.Equal(t, 6, len(snapshot1.ConnectedAgents))
	assert.Check(t, updated)

	delete(c.agents, agents[0].handler)

	snapshot2, updated := c.provisionerView.Update(c)

	assert.Equal(t, 0, len(snapshot2.IdleAgents))
	assert.Equal(t, 0, len(snapshot2.Tasks))
	assert.Equal(t, 5, len(snapshot2.ConnectedAgents))
	assert.Check(t, updated)
}

func TestTaskStateChange(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	pendingTask := addTask(t, system, c, "task1", 0, 1)

	snapshot1, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot1.IdleAgents))
	assert.Equal(t, 1, len(snapshot1.Tasks))
	assert.Equal(t, 6, len(snapshot1.ConnectedAgents))
	assert.Check(t, updated)

	c.reqList.SetAssignments(pendingTask.Handler, &ResourceAssigned{
		Assignments: make([]Assignment, 1),
	})

	snapshot2, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot2.IdleAgents))
	assert.Equal(t, 0, len(snapshot2.Tasks))
	assert.Equal(t, 6, len(snapshot2.ConnectedAgents))
	assert.Check(t, updated)
}

func TestTaskSlotsNeededChange(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)
	pendingTask := addTask(t, system, c, "task1", 0, 1)

	snapshot1, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot1.IdleAgents))
	assert.Equal(t, 1, len(snapshot1.Tasks))
	assert.Equal(t, 6, len(snapshot1.ConnectedAgents))
	assert.Check(t, updated)

	pendingTask.SlotsNeeded = 4

	snapshot2, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot2.IdleAgents))
	assert.Equal(t, 1, len(snapshot2.Tasks))
	assert.Equal(t, 6, len(snapshot2.ConnectedAgents))
	assert.Check(t, updated)
}

func TestAgentStateChange(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := createAgents(t, system, "agent", 1, 5)
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.provisionerView = newProvisionerView(4)

	snapshot1, updated := c.provisionerView.Update(c)

	assert.Equal(t, 1, len(snapshot1.IdleAgents))
	assert.Equal(t, 0, len(snapshot1.Tasks))
	assert.Equal(t, 6, len(snapshot1.ConnectedAgents))
	assert.Check(t, updated)

	i := 0
	for d := range agents[0].devices {
		if i == 0 {
			id := cproto.ID(uuid.New().String())
			agents[0].devices[d] = &id
		}
		i++
	}

	snapshot2, updated := c.provisionerView.Update(c)

	assert.Equal(t, 0, len(snapshot2.IdleAgents))
	assert.Equal(t, 0, len(snapshot2.Tasks))
	assert.Equal(t, 6, len(snapshot2.ConnectedAgents))
	assert.Check(t, updated)
}
