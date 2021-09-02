package resourcemanagers

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/google/uuid"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	cproto "github.com/determined-ai/determined/master/pkg/container"
)

func TestCleanUpTaskWhenTaskActorStopsWithError(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{{id: "agent", slots: 1}}
	tasks := []*mockTask{{id: "task", slotsNeeded: 1}}
	rp, ref := setupResourcePool(t, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetTaskSummaries{}).Get().(map[model.AllocationID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 1)

	system.Ask(taskRef, ThrowError{})
	assert.ErrorType(t, taskRef.StopAndAwaitTermination(), errMock)

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.len(), 0)
}

func TestCleanUpTaskWhenTaskActorPanics(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{{id: "agent", slots: 1}}
	tasks := []*mockTask{{id: "task", slotsNeeded: 1}}
	rp, ref := setupResourcePool(t, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetTaskSummaries{}).Get().(map[model.AllocationID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 1)

	system.Ask(taskRef, ThrowPanic{})
	assert.ErrorType(t, taskRef.StopAndAwaitTermination(), errMock)

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.len(), 0)
}

func TestCleanUpTaskWhenTaskActorStopsNormally(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{{id: "agent", slots: 1}}
	tasks := []*mockTask{{id: "task", slotsNeeded: 1}}
	rp, ref := setupResourcePool(t, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetTaskSummaries{}).Get().(map[model.AllocationID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 1)

	assert.NilError(t, taskRef.StopAndAwaitTermination())

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.len(), 0)
}

func TestCleanUpTaskWhenTaskActorReleaseResources(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{{id: "agent", slots: 1}}
	tasks := []*mockTask{{id: "task", slotsNeeded: 1}}
	rp, ref := setupResourcePool(t, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetTaskSummaries{}).Get().(map[model.AllocationID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 1)

	system.Ask(taskRef, sproto.ReleaseResources{}).Get()

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.len(), 0)
}

func TestScalingInfoAgentSummary(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{
		{id: "agent1", slots: 1},
		{id: "agent2", slots: 1},
	}
	tasks := []*mockTask{
		{id: "allocated-cpu-task1", slotsNeeded: 0, allocatedAgent: agents[0], containerStarted: true},
		{id: "allocated-cpu-task2", slotsNeeded: 0, allocatedAgent: agents[1], containerStarted: true},
		{id: "allocated-gpu-task3", slotsNeeded: 1, allocatedAgent: agents[1], containerStarted: true},
		{id: "unallocated-gpu-task4", slotsNeeded: 1},
		{id: "unallocated-gpu-task5", slotsNeeded: 5},
	}
	rp, _ := setupResourcePool(t, system, nil, tasks, nil, agents)
	rp.slotsPerInstance = 4

	// Test basic.
	updated := rp.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *rp.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent1": {Name: "agent1", IsIdle: false},
			"agent2": {Name: "agent2", IsIdle: false},
		},
	})

	// Test adding agents.
	agent3 := forceAddAgent(t, system, rp.agents, "agent3", 4, 0, 0)
	forceAddAgent(t, system, rp.agents, "agent4", 4, 1, 0)
	updated = rp.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *rp.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent1": {Name: "agent1", IsIdle: false},
			"agent2": {Name: "agent2", IsIdle: false},
			"agent3": {Name: "agent3", IsIdle: true},
			"agent4": {Name: "agent4", IsIdle: false},
		},
	})

	// Test removing agents.
	agent1 := system.Get(actor.Addr("agent1"))
	delete(rp.agents, agent1)
	updated = rp.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *rp.scalingInfo, sproto.ScalingInfo{
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
	for d := range rp.agents[agent3.handler].devices {
		if i == 0 {
			id := cproto.ID(uuid.New().String())
			rp.agents[agent3.handler].devices[d] = &id
		}
		i++
	}
	updated = rp.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *rp.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent2": {Name: "agent2", IsIdle: false},
			"agent3": {Name: "agent3", IsIdle: false},
			"agent4": {Name: "agent4", IsIdle: false},
		},
	})
}

func TestSettingGroupPriority(t *testing.T) {
	system := actor.NewSystem(t.Name())
	defaultPriority := 50
	config := ResourcePoolConfig{
		Scheduler: &SchedulerConfig{
			Priority: &PrioritySchedulerConfig{
				DefaultPriority: &defaultPriority,
			},
			FittingPolicy: best,
		},
	}

	rp, ref := setupResourcePool(t, system, &config, nil, nil, nil)

	// Test setting a non-default priority for a group.
	groupRefOne, created := system.ActorOf(actor.Addr("group1"), &mockGroup{})
	assert.Assert(t, created)
	updatedPriority := 22
	system.Tell(ref, sproto.SetGroupPriority{Priority: &updatedPriority, Handler: groupRefOne})

	// Test leaving the default priority for a group.
	groupRefTwo, created := system.ActorOf(actor.Addr("group2"), &mockGroup{})
	assert.Assert(t, created)
	system.Tell(ref, sproto.SetGroupPriority{Priority: nil, Handler: groupRefTwo})

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, *rp.groups[groupRefOne].priority, updatedPriority)
	assert.Equal(t, *rp.groups[groupRefTwo].priority, defaultPriority)
}
