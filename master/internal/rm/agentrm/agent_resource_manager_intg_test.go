//go:build integration
// +build integration

package agentrm

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestAgentRMRoutingTaskRelatedMessages(t *testing.T) {
	system := actor.NewSystem(t.Name())

	// This is required only due to the resource manager needing
	// to authenticate users when sending echo API requests.
	// No echo http requests are sent so it won't cause issues
	// initializing with nil values for this test.
	user.InitService(nil, nil, nil)

	// Set up one CPU resource pool and one GPU resource pool.
	cfg := &config.ResourceConfig{
		ResourceManager: &config.ResourceManagerConfig{
			AgentRM: &config.AgentResourceManagerConfig{
				Scheduler: &config.SchedulerConfig{
					FairShare:     &config.FairShareSchedulerConfig{},
					FittingPolicy: best,
				},
				DefaultAuxResourcePool:     "cpu-pool",
				DefaultComputeResourcePool: "gpu-pool",
			},
		},
		ResourcePools: []config.ResourcePoolConfig{
			{PoolName: "cpu-pool"},
			{PoolName: "gpu-pool"},
		},
	}
	_, cpuPoolRef := setupResourcePool(
		t, nil, system, &config.ResourcePoolConfig{PoolName: "cpu-pool"},
		nil, nil, []*MockAgent{{ID: "agent1", Slots: 0}},
	)
	_, gpuPoolRef := setupResourcePool(
		t, nil, system, &config.ResourcePoolConfig{PoolName: "gpu-pool"},
		nil, nil, []*MockAgent{{ID: "agent2", Slots: 4}},
	)
	agentRM := &agentResourceManager{
		config:      cfg.ResourceManager.AgentRM,
		poolsConfig: cfg.ResourcePools,
		pools: map[string]*actor.Ref{
			"cpu-pool": cpuPoolRef,
			"gpu-pool": gpuPoolRef,
		},
	}
	agentRMRef, created := system.ActorOf(actor.Addr("agentRM"), agentRM)
	assert.Assert(t, created)

	// Check if there are tasks.
	var taskSummaries map[model.AllocationID]sproto.AllocationSummary
	taskSummaries = system.Ask(
		agentRMRef, sproto.GetAllocationSummaries{}).
		Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 0)

	// Start CPU tasks actors
	var cpuTask1Ref, cpuTask2Ref *actor.Ref
	cpuTask1 := &MockTask{
		RMRef:        agentRMRef,
		ID:           "cpu-task1",
		SlotsNeeded:  0,
		ResourcePool: "cpu-pool",
	}
	cpuTask1Ref, created = system.ActorOf(actor.Addr(cpuTask1.ID), cpuTask1)
	assert.Assert(t, created)
	cpuTask2 := &MockTask{RMRef: agentRMRef, ID: "cpu-task2", SlotsNeeded: 0}
	cpuTask2Ref, created = system.ActorOf(actor.Addr(cpuTask2.ID), cpuTask2)
	assert.Assert(t, created)

	// Start GPU task actors.
	var gpuTask1Ref, gpuTask2Ref *actor.Ref
	gpuTask1 := &MockTask{
		RMRef:        agentRMRef,
		ID:           "gpu-task1",
		SlotsNeeded:  4,
		ResourcePool: "gpu-pool",
	}
	gpuTask1Ref, created = system.ActorOf(actor.Addr(gpuTask1.ID), gpuTask1)
	assert.Assert(t, created)
	gpuTask2 := &MockTask{RMRef: agentRMRef, ID: "gpu-task2", SlotsNeeded: 4}
	gpuTask2Ref, created = system.ActorOf(actor.Addr(gpuTask2.ID), gpuTask2)
	assert.Assert(t, created)

	// Let the CPU task actors request resources.
	system.Ask(cpuTask1Ref, SendRequestResourcesToResourceManager{}).Get()
	system.Ask(cpuTask2Ref, SendRequestResourcesToResourceManager{}).Get()

	// Check the resource pools of the tasks are correct.
	taskSummary := system.Ask(
		agentRMRef, sproto.GetAllocationSummary{ID: cpuTask1.ID}).Get().(*sproto.AllocationSummary)
	assert.Equal(t, taskSummary.ResourcePool, cpuTask1.ResourcePool)
	taskSummaries = system.Ask(
		agentRMRef, sproto.GetAllocationSummaries{}).
		Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(
		t,
		taskSummaries[cpuTask1.ID].ResourcePool,
		taskSummaries[cpuTask2.ID].ResourcePool,
	)

	// Let the GPU task actors request resources.
	system.Ask(gpuTask1Ref, SendRequestResourcesToResourceManager{}).Get()
	system.Ask(gpuTask2Ref, SendRequestResourcesToResourceManager{}).Get()

	// Check the resource pools of the tasks are correct.
	taskSummary = system.Ask(
		agentRMRef, sproto.GetAllocationSummary{ID: gpuTask1.ID}).Get().(*sproto.AllocationSummary)
	assert.Equal(t, taskSummary.ResourcePool, gpuTask1.ResourcePool)
	taskSummaries = system.Ask(
		agentRMRef, sproto.GetAllocationSummaries{}).
		Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(
		t,
		taskSummaries[gpuTask1.ID].ResourcePool,
		taskSummaries[gpuTask2.ID].ResourcePool,
	)

	// Let the CPU task actors release resources.
	system.Ask(cpuTask1Ref, SendResourcesReleasedToResourceManager{}).Get()
	system.Ask(cpuTask2Ref, SendResourcesReleasedToResourceManager{}).Get()
	taskSummaries = system.Ask(
		agentRMRef, sproto.GetAllocationSummaries{}).
		Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 2)

	// Let the GPU task actors release resources.
	system.Ask(gpuTask1Ref, SendResourcesReleasedToResourceManager{}).Get()
	system.Ask(gpuTask2Ref, SendResourcesReleasedToResourceManager{}).Get()
	taskSummaries = system.Ask(
		agentRMRef, sproto.GetAllocationSummaries{}).
		Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 0)

	// Fetch average queued time for resource pool
	db.MustSetupTestPostgres(t)
	_, err := agentRM.fetchAvgQueuedTime("cpu-pool")
	assert.NilError(t, err, "error fetch average queued time for cpu-pool")
	_, err = agentRM.fetchAvgQueuedTime("gpu-pool")
	assert.NilError(t, err, "error fetch average queued time for gpu-pool")
	_, err = agentRM.fetchAvgQueuedTime("non-existed-pool")
	assert.NilError(t, err, "error fetch average queued time for non-existed-pool")
}
