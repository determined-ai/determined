//go:build integration
// +build integration

package agentrm

import (
	"testing"

	"github.com/determined-ai/determined/master/internal/db"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
)

func TestAgentRMRoutingTaskRelatedMessages(t *testing.T) {
	// This is required only due to the resource manager needing
	// to authenticate users when sending echo API requests.
	// No echo http requests are sent so it won't cause issues
	// initializing with nil values for this test.
	user.InitService(nil, nil)

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
	cpuPoolRef := setupResourcePool(
		t, nil, &config.ResourcePoolConfig{PoolName: "cpu-pool"},
		nil, nil, []*MockAgent{{ID: "agent1", Slots: 0}},
	)
	gpuPoolRef := setupResourcePool(
		t, nil, &config.ResourcePoolConfig{PoolName: "gpu-pool"},
		nil, nil, []*MockAgent{{ID: "agent2", Slots: 4}},
	)
	agentRM := &ResourceManager{
		config:      cfg.ResourceManager.AgentRM,
		poolsConfig: cfg.ResourcePools,
		pools: map[string]*resourcePool{
			"cpu-pool": cpuPoolRef,
			"gpu-pool": gpuPoolRef,
		},
		agentUpdates: queue.New[agentUpdatedEvent](),
	}

	// Check if there are tasks.
	taskSummaries, err := agentRM.GetAllocationSummaries(sproto.GetAllocationSummaries{})
	require.NoError(t, err)
	assert.Equal(t, len(taskSummaries), 0)

	// Start CPU tasks actors
	cpuTask1 := &MockTask{
		ID:           "cpu-task1",
		SlotsNeeded:  0,
		ResourcePool: "cpu-pool",
	}
	cpuTask2 := &MockTask{ID: "cpu-task2", SlotsNeeded: 0}

	// Start GPU task actors.
	gpuTask1 := &MockTask{
		ID:           "gpu-task1",
		SlotsNeeded:  4,
		ResourcePool: "gpu-pool",
	}
	gpuTask2 := &MockTask{ID: "gpu-task2", SlotsNeeded: 4}

	// Let the CPU task actors request resources.
	_, err = agentRM.Allocate(sproto.AllocateRequest{
		AllocationID: cpuTask1.ID,
		SlotsNeeded:  cpuTask1.SlotsNeeded,
		ResourcePool: cpuTask1.ResourcePool,
	})
	require.NoError(t, err)
	_, err = agentRM.Allocate(sproto.AllocateRequest{
		AllocationID: cpuTask2.ID,
		SlotsNeeded:  cpuTask2.SlotsNeeded,
		ResourcePool: cpuTask2.ResourcePool,
	})
	require.NoError(t, err)

	// Check the resource pools of the tasks are correct.
	taskSummaries, err = agentRM.GetAllocationSummaries(sproto.GetAllocationSummaries{})
	require.NoError(t, err)
	assert.Equal(
		t,
		taskSummaries[cpuTask1.ID].ResourcePool,
		taskSummaries[cpuTask2.ID].ResourcePool,
	)

	// Let the GPU task actors request resources.
	_, err = agentRM.Allocate(sproto.AllocateRequest{
		AllocationID: gpuTask1.ID,
		SlotsNeeded:  gpuTask1.SlotsNeeded,
		ResourcePool: gpuTask1.ResourcePool,
	})
	require.NoError(t, err)
	_, err = agentRM.Allocate(sproto.AllocateRequest{
		AllocationID: gpuTask2.ID,
		SlotsNeeded:  gpuTask2.SlotsNeeded,
		ResourcePool: gpuTask2.ResourcePool,
	})
	require.NoError(t, err)

	// Check the resource pools of the tasks are correct.
	taskSummaries, err = agentRM.GetAllocationSummaries(sproto.GetAllocationSummaries{})
	require.NoError(t, err)
	assert.Equal(
		t,
		taskSummaries[gpuTask1.ID].ResourcePool,
		taskSummaries[gpuTask2.ID].ResourcePool,
	)

	// Let the CPU task actors release resources.
	agentRM.Release(
		sproto.ResourcesReleased{
			AllocationID: cpuTask1.ID,
			ResourcePool: taskSummaries[cpuTask1.ID].ResourcePool,
		},
	)
	agentRM.Release(sproto.ResourcesReleased{
		AllocationID: cpuTask2.ID,
		ResourcePool: taskSummaries[cpuTask2.ID].ResourcePool,
	})
	taskSummaries, err = agentRM.GetAllocationSummaries(sproto.GetAllocationSummaries{})
	require.NoError(t, err)
	assert.Equal(t, len(taskSummaries), 2)

	// Let the GPU task actors release resources.
	agentRM.Release(sproto.ResourcesReleased{
		AllocationID: gpuTask1.ID,
		ResourcePool: taskSummaries[gpuTask1.ID].ResourcePool,
	})
	agentRM.Release(sproto.ResourcesReleased{
		AllocationID: gpuTask2.ID,
		ResourcePool: taskSummaries[gpuTask2.ID].ResourcePool,
	})
	taskSummaries, err = agentRM.GetAllocationSummaries(sproto.GetAllocationSummaries{})
	require.NoError(t, err)
	assert.Equal(t, len(taskSummaries), 0)

	// Fetch average queued time for resource pool
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, "file://../../../static/migrations")
	_, err = agentRM.fetchAvgQueuedTime("cpu-pool")
	assert.NilError(t, err, "error fetch average queued time for cpu-pool")
	_, err = agentRM.fetchAvgQueuedTime("gpu-pool")
	assert.NilError(t, err, "error fetch average queued time for gpu-pool")
	_, err = agentRM.fetchAvgQueuedTime("non-existed-pool")
	assert.NilError(t, err, "error fetch average queued time for non-existed-pool")
}
