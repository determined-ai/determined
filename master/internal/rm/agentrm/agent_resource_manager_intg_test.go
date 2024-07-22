//go:build integration
// +build integration

package agentrm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

func TestAgentRMRoutingTaskRelatedMessages(t *testing.T) {
	// This is required only due to the resource manager needing
	// to authenticate users when sending echo API requests.
	// No echo http requests are sent so it won't cause issues
	// initializing with nil values for this test.
	user.InitService(nil, nil)

	// Set up one CPU resource pool and one GPU resource pool.
	cfg := &config.ResourceConfig{
		RootManagerInternal: &config.ResourceManagerConfig{
			AgentRM: &config.AgentResourceManagerConfig{
				Scheduler: &config.SchedulerConfig{
					FairShare:     &config.FairShareSchedulerConfig{},
					FittingPolicy: best,
				},
				DefaultAuxResourcePool:     "cpu-pool",
				DefaultComputeResourcePool: "gpu-pool",
			},
		},
		RootPoolsInternal: []config.ResourcePoolConfig{
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
		config:      cfg.ResourceManagers()[0].ResourceManager.AgentRM,
		poolsConfig: cfg.ResourceManagers()[0].ResourcePools,
		pools: map[string]*resourcePool{
			"cpu-pool": cpuPoolRef,
			"gpu-pool": gpuPoolRef,
		},
		agentUpdates: queue.New[agentUpdatedEvent](),
	}

	// Check if there are tasks.
	taskSummaries, err := agentRM.GetAllocationSummaries()
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
	taskSummaries, err = agentRM.GetAllocationSummaries()
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
	taskSummaries, err = agentRM.GetAllocationSummaries()
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
	taskSummaries, err = agentRM.GetAllocationSummaries()
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
	taskSummaries, err = agentRM.GetAllocationSummaries()
	require.NoError(t, err)
	assert.Equal(t, len(taskSummaries), 0)

	// Fetch average queued time for resource pool
	_, err = agentRM.fetchAvgQueuedTime("cpu-pool")
	assert.NilError(t, err, "error fetch average queued time for cpu-pool")
	_, err = agentRM.fetchAvgQueuedTime("gpu-pool")
	assert.NilError(t, err, "error fetch average queued time for gpu-pool")
	_, err = agentRM.fetchAvgQueuedTime("non-existed-pool")
	assert.NilError(t, err, "error fetch average queued time for non-existed-pool")
}

func TestGetResourcePools(t *testing.T) {
	expectedName := "testname"
	expectedMetadata := map[string]string{"x": "y*y"}
	cfg := &config.ResourceConfig{
		RootManagerInternal: &config.ResourceManagerConfig{
			AgentRM: &config.AgentResourceManagerConfig{
				ClusterName: expectedName,
				Metadata:    expectedMetadata,
				Scheduler: &config.SchedulerConfig{
					FairShare:     &config.FairShareSchedulerConfig{},
					FittingPolicy: best,
				},
				DefaultAuxResourcePool:     "cpu-pool",
				DefaultComputeResourcePool: "gpu-pool",
			},
		},
		RootPoolsInternal: []config.ResourcePoolConfig{
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
		config:      cfg.ResourceManagers()[0].ResourceManager.AgentRM,
		poolsConfig: cfg.ResourceManagers()[0].ResourcePools,
		pools: map[string]*resourcePool{
			"cpu-pool": cpuPoolRef,
			"gpu-pool": gpuPoolRef,
		},
		agentUpdates: queue.New[agentUpdatedEvent](),
	}

	resp, err := agentRM.GetResourcePools()
	require.NoError(t, err)
	actual, err := json.MarshalIndent(resp.ResourcePools, "", "  ")
	require.NoError(t, err)

	expectedPools := []*resourcepoolv1.ResourcePool{
		{
			Name:                    "cpu-pool",
			Type:                    resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_STATIC,
			DefaultAuxPool:          true,
			SlotsPerAgent:           -1,
			SchedulerType:           resourcepoolv1.SchedulerType_SCHEDULER_TYPE_FAIR_SHARE,
			SchedulerFittingPolicy:  resourcepoolv1.FittingPolicy_FITTING_POLICY_BEST,
			Location:                "on-prem",
			Details:                 &resourcepoolv1.ResourcePoolDetail{},
			Stats:                   &jobv1.QueueStats{},
			ClusterName:             expectedName,
			ResourceManagerMetadata: expectedMetadata,
		},
		{
			Name:                    "gpu-pool",
			Type:                    resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_STATIC,
			DefaultComputePool:      true,
			SlotsPerAgent:           -1,
			SchedulerType:           resourcepoolv1.SchedulerType_SCHEDULER_TYPE_FAIR_SHARE,
			SchedulerFittingPolicy:  resourcepoolv1.FittingPolicy_FITTING_POLICY_BEST,
			Location:                "on-prem",
			Details:                 &resourcepoolv1.ResourcePoolDetail{},
			Stats:                   &jobv1.QueueStats{},
			ClusterName:             expectedName,
			ResourceManagerMetadata: expectedMetadata,
		},
	}
	expected, err := json.MarshalIndent(expectedPools, "", "  ")
	require.NoError(t, err)

	require.Equal(t, string(expected), string(actual))
}

func TestGetJobQueueStatsRequest(t *testing.T) {
	agentRM := &ResourceManager{
		pools: map[string]*resourcePool{
			"pool1": setupResourcePool(
				t, nil, &config.ResourcePoolConfig{PoolName: "pool1"},
				nil, nil, []*MockAgent{{ID: "agent1", Slots: 0}},
			),
			"pool2": setupResourcePool(
				t, nil, &config.ResourcePoolConfig{PoolName: "pool2"},
				nil, nil, []*MockAgent{{ID: "agent2", Slots: 0}},
			),
		},
	}

	cases := []struct {
		name        string
		filteredRPs []string
		expected    int
	}{
		{"empty, return all", []string{}, 2},
		{"filter 1 in", []string{"pool1"}, 1},
		{"filter 2 in", []string{"pool1", "pool2"}, 2},
		{"filter undefined in, return none", []string{"bogus"}, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := agentRM.GetJobQueueStatsRequest(&apiv1.GetJobQueueStatsRequest{ResourcePools: tt.filteredRPs})
			require.NoError(t, err)
			require.Len(t, res.Results, tt.expected)
		})
	}
}
