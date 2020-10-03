package resourcemanagers

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/google/uuid"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	cproto "github.com/determined-ai/determined/master/pkg/container"
)

func (rp *ResourcePool) addAllocatedTask(
	req *AllocateRequest, allocated *ResourcesAllocated,
) {
	rp.taskList.AddTask(req)
	rp.taskList.SetAllocations(req.TaskActor, allocated)
}

func (rp *ResourcePool) addAgent(
	t *testing.T,
	system *actor.System,
	agentID string,
	numSlots int,
	numUsedSlots int,
	numZeroSlotContainers int,
) *agentState {
	agent := createAgent(t, system, agentID, numSlots, numUsedSlots, numZeroSlotContainers)
	rp.agents[agent.handler] = agent
	return agent
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

func setupCluster(
	scheduler Scheduler, fittingMethod SoftConstraint, agents []*agentState, tasks []*actor.Ref,
) *ResourcePool {
	d := ResourcePool{
		config:        &ResourcePoolConfig{},
		scheduler:     scheduler,
		fittingMethod: fittingMethod,
		agents:        make(map[*actor.Ref]*agentState),
		groups:        make(map[*actor.Ref]*group),

		taskList:    newTaskList(),
		scalingInfo: &sproto.ScalingInfo{},

		reschedule: false,
	}

	for _, agent := range agents {
		d.agents[agent.handler] = agent
	}

	for _, handler := range tasks {
		system := handler.System()

		g := system.Ask(handler, GetGroup{}).Get().(*actor.Ref)
		slots := system.Ask(handler, GetSlots{}).Get().(int)
		label := system.Ask(handler, GetLabel{}).Get().(string)

		d.addAllocatedTask(&AllocateRequest{
			ID:          TaskID(handler.Address().String()),
			Name:        handler.Address().Local(),
			Group:       g,
			TaskActor:   handler,
			SlotsNeeded: slots,
			Label:       label,
		}, nil)
		_ = d.getOrCreateGroup(nil, g)
		if resp := system.Ask(g, getMaxSlots{}); resp.Get() != nil {
			d.getOrCreateGroup(nil, g).maxSlots = resp.Get().(*int)
		}
		if resp := system.Ask(g, getWeight{}); resp.Get() != nil {
			d.getOrCreateGroup(nil, g).weight = resp.Get().(float64)
		}
	}
	return &d
}

func TestCleanUpTaskWhenTaskActorStopsWithError(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, "agent", 1, "")}
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, nil)
	c.saveNotifications = true
	cluster, created := system.ActorOf(actor.Addr("scheduler"), c)
	assert.Assert(t, created)
	mockActor, created := system.ActorOf(
		actor.Addr("mockTaskActor"),
		&mockTask{
			cluster: cluster,
			system:  system,
		},
	)
	assert.Assert(t, created)

	system.Ask(mockActor, AskSchedulerToAddTask{
		task: AllocateRequest{
			ID:          TaskID(uuid.New().String()),
			Name:        "mock_task",
			Group:       mockActor,
			SlotsNeeded: 1,
		},
	}).Get()
	assert.Equal(t, c.taskList.len(), 1)

	system.Ask(mockActor, ThrowError{})
	assert.ErrorType(t, mockActor.StopAndAwaitTermination(), errMock)

	for _, c := range c.notifications {
		<-c
	}

	assert.NilError(t, cluster.StopAndAwaitTermination())
	assert.Equal(t, c.taskList.len(), 0)
}

func TestCleanUpTaskWhenTaskActorPanics(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, "agent", 1, "")}
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, nil)
	c.saveNotifications = true
	cluster, created := system.ActorOf(actor.Addr("scheduler"), c)
	assert.Assert(t, created)
	mockActor, created := system.ActorOf(
		actor.Addr("mockTaskActor"),
		&mockTask{
			cluster: cluster,
			system:  system,
		},
	)
	assert.Assert(t, created)

	system.Ask(mockActor, AskSchedulerToAddTask{
		task: AllocateRequest{
			ID:          TaskID(uuid.New().String()),
			Name:        "mock_task",
			Group:       mockActor,
			SlotsNeeded: 1,
		},
	}).Get()

	assert.Equal(t, c.taskList.len(), 1)
	system.Ask(mockActor, ThrowPanic{})
	assert.ErrorType(t, mockActor.StopAndAwaitTermination(), errMock)

	for _, c := range c.notifications {
		<-c
	}

	assert.NilError(t, cluster.StopAndAwaitTermination())
	assert.Equal(t, c.taskList.len(), 0)
}

func TestCleanUpTaskWhenTaskActorStopsNormally(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, "agent", 1, "")}
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, nil)
	c.saveNotifications = true
	cluster, created := system.ActorOf(actor.Addr("scheduler"), c)
	assert.Assert(t, created)

	mockActor, created := system.ActorOf(
		actor.Addr("mockTaskActor"),
		&mockTask{
			cluster: cluster,
			system:  system,
		},
	)
	assert.Assert(t, created)

	system.Ask(mockActor, AskSchedulerToAddTask{
		task: AllocateRequest{
			ID:          TaskID(uuid.New().String()),
			Name:        "mock_task",
			Group:       mockActor,
			SlotsNeeded: 1,
		},
	}).Get()

	assert.Equal(t, c.taskList.len(), 1)

	assert.NilError(t, mockActor.StopAndAwaitTermination())

	for _, c := range c.notifications {
		<-c
	}

	assert.NilError(t, cluster.StopAndAwaitTermination())
	assert.Equal(t, c.taskList.len(), 0)
}

func testWhenActorsStopOrTaskIsKilled(t *testing.T, r *rand.Rand) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{newMockAgent(t, system, fmt.Sprintf("agent-%d", r.Int()), 1, "")}
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, nil)
	cluster, created := system.ActorOf(actor.Addr("scheduler"), c)
	assert.Assert(t, created)

	mockActor, created := system.ActorOf(
		actor.Addr("mockTaskActor"),
		&mockTask{
			cluster: cluster,
			system:  system,
		})
	assert.Assert(t, created)

	system.Ask(mockActor, AskSchedulerToAddTask{
		task: AllocateRequest{
			ID:          TaskID(uuid.New().String()),
			Name:        "mock_task",
			Group:       mockActor,
			SlotsNeeded: 1,
		},
	}).Get()

	actions := []func(){
		func() {
			system.Tell(cluster, ResourcesReleased{
				TaskActor: mockActor,
			})
		},
		func() {
			system.Tell(cluster, sproto.RemoveAgent{
				Agent: agents[0].handler,
			})
		},
	}

	r.Shuffle(len(actions), func(i, j int) {
		actions[i], actions[j] = actions[j], actions[i]
	})

	for _, fn := range actions {
		fn()
	}

	assert.NilError(t, cluster.StopAndAwaitTermination())
	assert.Equal(t, c.taskList.len(), 0)
}

func TestCleanUpTaskWhenActorsStopOrTaskIsKilled(t *testing.T) {
	r := rand.New(rand.NewSource(0))

	// When the actor messages are actually processed is non-deterministic,
	// re-run this test a couple times to ensure interesting interleavings.
	for i := 0; i < 10; i++ {
		testWhenActorsStopOrTaskIsKilled(t, r)
	}
}

func TestScalingInfoAgentSummary(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{
		createAgent(t, system, "agent1", 1, 0, 1),
		createAgent(t, system, "agent2", 1, 1, 1),
	}
	var tasks []*actor.Ref
	c := setupCluster(NewFairShareScheduler(), BestFit, agents, tasks)
	c.slotsPerInstance = 4

	addTask(t, system, c.taskList, "task1", 1, 1)
	addTask(t, system, c.taskList, "task2", 0, 1)
	addTask(t, system, c.taskList, "task3", 0, 5)

	// Test basic.
	updated := c.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *c.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent1": {Name: "agent1", IsIdle: false},
			"agent2": {Name: "agent2", IsIdle: false},
		},
	})

	// Test adding agents.
	agent3 := c.addAgent(t, system, "agent3", 4, 0, 0)
	c.addAgent(t, system, "agent4", 4, 1, 0)
	updated = c.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *c.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent1": {Name: "agent1", IsIdle: false},
			"agent2": {Name: "agent2", IsIdle: false},
			"agent3": {Name: "agent3", IsIdle: true},
			"agent4": {Name: "agent4", IsIdle: false},
		},
	})

	// Test removing agents.
	delete(c.agents, agents[0].handler)
	updated = c.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *c.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent2": {Name: "agent2", IsIdle: false},
			"agent3": {Name: "agent3", IsIdle: true},
			"agent4": {Name: "agent4", IsIdle: false},
		},
	})

	// Test agent state change.
	// Allocate a container to a device of the agent2.
	i := 0
	for d := range c.agents[agent3.handler].devices {
		if i == 0 {
			id := cproto.ID(uuid.New().String())
			c.agents[agent3.handler].devices[d] = &id
		}
		i++
	}
	updated = c.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *c.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent2": {Name: "agent2", IsIdle: false},
			"agent3": {Name: "agent3", IsIdle: false},
			"agent4": {Name: "agent4", IsIdle: false},
		},
	})
}
