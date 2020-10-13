package resourcemanagers

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestAgentRMTaskRouting(t *testing.T) {
	system := actor.NewSystem(t.Name())

	// Set up one CPU resource pool and one GPU resource pool.
	rmConfig := &AgentResourceManagerConfig{
		SchedulingPolicy:       "fair_share",
		FittingPolicy:          "best",
		DefaultCPUResourcePool: "cpu-pool",
		DefaultGPUResourcePool: "gpu-pool",
	}
	poolsConfig := &ResourcePoolsConfig{
		ResourcePools: []ResourcePoolConfig{
			{PoolName: "cpu-pool"},
			{PoolName: "gpu-pool"},
		},
	}
	_, cpuPoolRef := setupResourcePool(
		t, system, &ResourcePoolConfig{PoolName: "cpu-pool"},
		nil, nil, []*mockAgent{{id: "agent1", slots: 0}},
	)
	_, gpuPoolRef := setupResourcePool(
		t, system, &ResourcePoolConfig{PoolName: "gpu-pool"},
		nil, nil, []*mockAgent{{id: "agent2", slots: 4}},
	)
	agentRM := &agentResourceManager{
		config:      rmConfig,
		poolsConfig: poolsConfig,
		pools: map[string]*actor.Ref{
			"cpu-pool": cpuPoolRef,
			"gpu-pool": gpuPoolRef,
		},
	}
	agentRMRef, created := system.ActorOf(actor.Addr("agentRM"), agentRM)
	assert.Assert(t, created)

	// Add one CPU task.
	cpuTask := &mockTask{rmRef: agentRMRef, id: "cpu-task", slotsNeeded: 0, resourcePool: "cpu-pool"}
	cpuTaskRef, created := system.ActorOf(actor.Addr(cpuTask.id), cpuTask)
	assert.Assert(t, created)

	// Add one GPU task.
	gpuTask := &mockTask{rmRef: agentRMRef, id: "gpu-task", slotsNeeded: 4, resourcePool: "gpu-pool"}
	gpuTaskRef, created := system.ActorOf(actor.Addr(gpuTask.id), gpuTask)
	assert.Assert(t, created)

	var taskSummaries map[TaskID]TaskSummary
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 0)

	// Let the CPU task to request resources.
	system.Ask(cpuTaskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummary := system.Ask(agentRMRef, GetTaskSummary{ID: &cpuTask.id}).Get().(*TaskSummary)
	assert.Equal(t, taskSummary.ResourcePool, cpuTask.resourcePool)
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, taskSummaries[cpuTask.id].ResourcePool, cpuTask.resourcePool)

	// Let the GPU task to request resources.
	system.Ask(gpuTaskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummary = system.Ask(agentRMRef, GetTaskSummary{ID: &gpuTask.id}).Get().(*TaskSummary)
	assert.Equal(t, taskSummary.ResourcePool, gpuTask.resourcePool)
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, taskSummaries[gpuTask.id].ResourcePool, gpuTask.resourcePool)

	// Let the CPU task to release resources.
	system.Ask(cpuTaskRef, SendResourcesReleasedToResourceManager{}).Get()
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 1)

	// Let the GPU task to release resources.
	system.Ask(gpuTaskRef, SendResourcesReleasedToResourceManager{}).Get()
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 0)
}
