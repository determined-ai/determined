package resourcemanagers

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestAgentRMRoutingTaskRelatedMessages(t *testing.T) {
	system := actor.NewSystem(t.Name())

	// Set up one CPU resource pool and one GPU resource pool.
	config := &ResourceConfig{
		ResourceManager: &ResourceManagerConfig{
			AgentRM: &AgentResourceManagerConfig{
				Scheduler: &SchedulerConfig{
					FairShare:     &FairShareSchedulerConfig{},
					FittingPolicy: defaultFitPolicy,
				},
				DefaultCPUResourcePool: "cpu-pool",
				DefaultGPUResourcePool: "gpu-pool",
			},
		},
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
		config:      config.ResourceManager.AgentRM,
		poolsConfig: config.ResourcePools,
		pools: map[string]*actor.Ref{
			"cpu-pool": cpuPoolRef,
			"gpu-pool": gpuPoolRef,
		},
	}
	agentRMRef, created := system.ActorOf(actor.Addr("agentRM"), agentRM)
	assert.Assert(t, created)

	// Check if there are tasks.
	var taskSummaries map[TaskID]TaskSummary
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 0)

	// Start CPU tasks actors
	var cpuTask1Ref, cpuTask2Ref *actor.Ref
	cpuTask1 := &mockTask{rmRef: agentRMRef, id: "cpu-task1", slotsNeeded: 0, resourcePool: "cpu-pool"}
	cpuTask1Ref, created = system.ActorOf(actor.Addr(cpuTask1.id), cpuTask1)
	assert.Assert(t, created)
	cpuTask2 := &mockTask{rmRef: agentRMRef, id: "cpu-task2", slotsNeeded: 0}
	cpuTask2Ref, created = system.ActorOf(actor.Addr(cpuTask2.id), cpuTask2)
	assert.Assert(t, created)

	// Start GPU task actors.
	var gpuTask1Ref, gpuTask2Ref *actor.Ref
	gpuTask1 := &mockTask{rmRef: agentRMRef, id: "gpu-task1", slotsNeeded: 4, resourcePool: "gpu-pool"}
	gpuTask1Ref, created = system.ActorOf(actor.Addr(gpuTask1.id), gpuTask1)
	assert.Assert(t, created)
	gpuTask2 := &mockTask{rmRef: agentRMRef, id: "gpu-task2", slotsNeeded: 4}
	gpuTask2Ref, created = system.ActorOf(actor.Addr(gpuTask2.id), gpuTask2)
	assert.Assert(t, created)

	// Let the CPU task actors request resources.
	system.Ask(cpuTask1Ref, SendRequestResourcesToResourceManager{}).Get()
	system.Ask(cpuTask2Ref, SendRequestResourcesToResourceManager{}).Get()

	// Check the resource pools of the tasks are correct.
	taskSummary := system.Ask(agentRMRef, GetTaskSummary{ID: &cpuTask1.id}).Get().(*TaskSummary)
	assert.Equal(t, taskSummary.ResourcePool, cpuTask1.resourcePool)
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, taskSummaries[cpuTask1.id].ResourcePool, taskSummaries[cpuTask2.id].ResourcePool)

	// Let the GPU task actors request resources.
	system.Ask(gpuTask1Ref, SendRequestResourcesToResourceManager{}).Get()
	system.Ask(gpuTask2Ref, SendRequestResourcesToResourceManager{}).Get()

	// Check the resource pools of the tasks are correct.
	taskSummary = system.Ask(agentRMRef, GetTaskSummary{ID: &gpuTask1.id}).Get().(*TaskSummary)
	assert.Equal(t, taskSummary.ResourcePool, gpuTask1.resourcePool)
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, taskSummaries[gpuTask1.id].ResourcePool, taskSummaries[gpuTask2.id].ResourcePool)

	// Let the CPU task actors release resources.
	system.Ask(cpuTask1Ref, SendResourcesReleasedToResourceManager{}).Get()
	system.Ask(cpuTask2Ref, SendResourcesReleasedToResourceManager{}).Get()
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 2)

	// Let the GPU task actors release resources.
	system.Ask(gpuTask1Ref, SendResourcesReleasedToResourceManager{}).Get()
	system.Ask(gpuTask2Ref, SendResourcesReleasedToResourceManager{}).Get()
	taskSummaries = system.Ask(agentRMRef, GetTaskSummaries{}).Get().(map[TaskID]TaskSummary)
	assert.Equal(t, len(taskSummaries), 0)
}
