package agentrm

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

// A lot of these tests don't make sense anymore post actor. I refactored them shoddily because I know what the
// test is already covered by allocation tests. We should circle back and write better tests.

func TestCleanUpTaskWhenTaskActorStopsWithError(t *testing.T) {
	agents := []*MockAgent{{ID: "agent", Slots: 1}}
	tasks := []*MockTask{{ID: "task", SlotsNeeded: 1}}
	rp := setupResourcePool(t, nil, nil, tasks, nil, agents)

	rp.Allocate(sproto.AllocateRequest{AllocationID: tasks[0].ID, SlotsNeeded: tasks[0].SlotsNeeded})
	taskSummaries := rp.GetAllocationSummaries()
	assert.Equal(t, len(taskSummaries), 1)

	rp.ResourcesReleased(sproto.ResourcesReleased{
		AllocationID: tasks[0].ID,
		ResourcePool: tasks[0].ResourcePool,
	})

	for _, n := range rp.notifications {
		<-n
	}

	rp.stop()
	assert.Equal(t, len(rp.GetAllocationSummaries()), 0)
}

func TestCleanUpTaskWhenTaskActorPanics(t *testing.T) {
	agents := []*MockAgent{{ID: "agent", Slots: 1}}
	tasks := []*MockTask{{ID: "task", SlotsNeeded: 1}}
	rp := setupResourcePool(t, nil, nil, tasks, nil, agents)

	rp.Allocate(sproto.AllocateRequest{AllocationID: tasks[0].ID, SlotsNeeded: tasks[0].SlotsNeeded})
	taskSummaries := rp.GetAllocationSummaries()
	assert.Equal(t, len(taskSummaries), 1)

	rp.ResourcesReleased(sproto.ResourcesReleased{
		AllocationID: tasks[0].ID,
		ResourcePool: tasks[0].ResourcePool,
	})

	for _, n := range rp.notifications {
		<-n
	}

	rp.stop()
	assert.Equal(t, len(rp.GetAllocationSummaries()), 0)
}

func TestCleanUpTaskWhenTaskActorStopsNormally(t *testing.T) {
	agents := []*MockAgent{{ID: "agent", Slots: 1}}
	tasks := []*MockTask{{ID: "task", SlotsNeeded: 1}}
	rp := setupResourcePool(t, nil, nil, tasks, nil, agents)

	rp.Allocate(sproto.AllocateRequest{AllocationID: tasks[0].ID, SlotsNeeded: tasks[0].SlotsNeeded})
	taskSummaries := rp.GetAllocationSummaries()
	assert.Equal(t, len(taskSummaries), 1)

	rp.ResourcesReleased(sproto.ResourcesReleased{
		AllocationID: tasks[0].ID,
		ResourcePool: tasks[0].ResourcePool,
	})

	for _, n := range rp.notifications {
		<-n
	}

	rp.stop()
	assert.Equal(t, len(rp.GetAllocationSummaries()), 0)
}

func TestCleanUpTaskWhenTaskActorReleaseResources(t *testing.T) {
	agents := []*MockAgent{{ID: "agent", Slots: 1}}
	tasks := []*MockTask{{ID: "task", SlotsNeeded: 1}}
	rp := setupResourcePool(t, nil, nil, tasks, nil, agents)

	rp.Allocate(sproto.AllocateRequest{AllocationID: tasks[0].ID, SlotsNeeded: tasks[0].SlotsNeeded})
	taskSummaries := rp.GetAllocationSummaries()
	assert.Equal(t, len(taskSummaries), 1)

	rp.ResourcesReleased(sproto.ResourcesReleased{
		AllocationID: tasks[0].ID,
		ResourcePool: tasks[0].ResourcePool,
	})

	rp.stop()
	assert.Equal(t, len(rp.GetAllocationSummaries()), 0)
}

func TestScalingInfoAgentSummary(t *testing.T) {
	agents := []*MockAgent{
		{ID: "agent1", Slots: 1},
		{ID: "agent2", Slots: 1},
	}
	tasks := []*MockTask{
		{
			ID:               "allocated-cpu-task1",
			SlotsNeeded:      0,
			AllocatedAgent:   agents[0],
			ContainerStarted: true,
		},
		{
			ID:               "allocated-cpu-task2",
			SlotsNeeded:      0,
			AllocatedAgent:   agents[1],
			ContainerStarted: true,
		},
		{
			ID:               "allocated-gpu-task3",
			SlotsNeeded:      1,
			AllocatedAgent:   agents[1],
			ContainerStarted: true,
		},
		{ID: "unallocated-gpu-task4", SlotsNeeded: 1},
		{ID: "unallocated-gpu-task5", SlotsNeeded: 5},
	}
	rp := setupResourcePool(t, nil, nil, tasks, nil, agents)
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
	agent3 := forceAddAgent(t, rp.agentStatesCache, "agent3", 4, 0, 0)
	forceAddAgent(t, rp.agentStatesCache, "agent4", 4, 1, 0)
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
	delete(rp.agentStatesCache, aproto.ID("agent1"))
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
	for d := range rp.agentStatesCache[agent3.id].Devices {
		if i == 0 {
			id := cproto.ID(uuid.New().String())
			rp.agentStatesCache[agent3.id].Devices[d] = &id
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
	defaultPriority := 50
	config := config.ResourcePoolConfig{
		Scheduler: &config.SchedulerConfig{
			Priority: &config.PrioritySchedulerConfig{
				DefaultPriority: &defaultPriority,
			},
			FittingPolicy: best,
		},
	}

	rp := setupResourcePool(t, nil, &config, nil, nil, nil)

	// Test setting a non-default priority for a group.
	updatedPriority := 22
	jobID := model.NewJobID()
	assert.Equal(t, tasklist.GroupPriorityChangeRegistry.Add(jobID, nil), nil)
	err := rp.SetGroupPriority(sproto.SetGroupPriority{Priority: updatedPriority, JobID: jobID})
	require.NoError(t, err)

	for _, n := range rp.notifications {
		<-n
	}

	rp.stop()
	assert.Check(t, rp.groups[jobID] != nil)
	assert.Check(t, rp.groups[jobID].Priority != nil)
	assert.Equal(t, *rp.groups[jobID].Priority, updatedPriority)
	assert.Equal(t, tasklist.GroupPriorityChangeRegistry.Delete(jobID), nil)

	time.Sleep(time.Second)
	rp.mu.Lock()
	assert.Check(t, rp.groups[jobID] == nil)
	rp.mu.Unlock()
}

func TestGetResourceSummary(t *testing.T) {
	defaultPriority := 50
	config := config.ResourcePoolConfig{
		Scheduler: &config.SchedulerConfig{
			Priority: &config.PrioritySchedulerConfig{
				DefaultPriority: &defaultPriority,
			},
			FittingPolicy: best,
		},
	}

	cases := []struct {
		name    string
		devices device.Type
		agents  []*MockAgent
	}{
		{"simple", device.Type(""), nil},
		{"homogenous agents", device.Type("cuda"), []*MockAgent{
			{ID: "agent1", SlotType: "cuda", Slots: 1},
			{ID: "agent2", SlotType: "cuda", Slots: 2},
		}},
		{"heterogenous agents", device.Type(""), []*MockAgent{
			{ID: "agent1", SlotType: "cuda"},
			{ID: "agent2", SlotType: "cpu"},
			{ID: "agent2", SlotType: "rocm"},
		}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rp := setupResourcePool(t, nil, &config, nil, nil, tt.agents)
			summary := rp.resourceSummaryFromAgentStates(rp.agentStatesCache)
			require.Equal(t, tt.devices, summary.slotType)
		})
	}
}
