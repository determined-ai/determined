package dispatcherrm

import (
	"reflect"
	"testing"

	"gotest.tools/assert"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const launcherPoolDescription = "launcher-pool-1-description"

func Test_generateGetAgentsResponse(t *testing.T) {
	n1 := hpcNodeDetails{
		Partitions:    []string{"Partition 1"},
		Addresses:     []string{"address 1", "address 2"},
		Draining:      true,
		Allocated:     true,
		Name:          "Node 1",
		GpuCount:      0,
		GpuInUseCount: 0,
		CPUCount:      8,
		CPUInUseCount: 6,
	}

	n2 := hpcNodeDetails{
		Partitions:    []string{"NoOverride", "Partition 1", "Partition 2"},
		Addresses:     []string{"address 3", "address 4"},
		Draining:      false,
		Allocated:     true,
		Name:          "Node 2",
		GpuCount:      2,
		GpuInUseCount: 1,
		CPUCount:      8,
		CPUInUseCount: 0,
	}

	n3 := hpcNodeDetails{
		Partitions:    []string{"Partition 1", "Partition 3"},
		Addresses:     []string{"address 3", "address 4"},
		Draining:      false,
		Allocated:     true,
		Name:          "Node 3",
		GpuCount:      2,
		GpuInUseCount: 1,
		CPUCount:      8,
		CPUInUseCount: 0,
	}

	nodes := []hpcNodeDetails{n1, n2, n3}

	hpcResource := &hpcResources{
		Nodes: nodes,
	}

	rocm := device.ROCM
	cuda := device.CUDA
	overrides := map[string]config.DispatcherPartitionOverrideConfigs{
		"Partition 2": {
			SlotType: &rocm,
		},
		"Partition 3": {
			SlotType: &cuda,
		},
	}

	// State that "Partition 2" is the providing partition for "launcher-provided-pool"
	poolProviderMap := map[string][]string{
		"Partition 2": {"launcher-provided-pool"},
	}
	config := &config.DispatcherResourceManagerConfig{
		PartitionOverrides: overrides,
	}

	// Expect each agent to participate in resource pools as follows:
	expectedResourcePools := [][]string{
		{"Partition 1"},
		{"NoOverride", "Partition 1", "Partition 2", "launcher-provided-pool"},
		{"Partition 1", "Partition 3"},
	}

	m := &DispatcherResourceManager{
		rmConfig:        config,
		hpcDetailsCache: makeTestHpcDetailsCache(hpcResource),
		poolProviderMap: poolProviderMap,
		dbState:         *newDispatcherState(),
	}

	want0 := map[string]*agentv1.Slot{
		"/agents/Node 1/slots/0": {
			Id:        "0",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_CPU},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING},
		},
		"/agents/Node 1/slots/1": {
			Id:        "1",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_CPU},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING},
		},
		"/agents/Node 1/slots/2": {
			Id:        "2",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_CPU},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING},
		},
		"/agents/Node 1/slots/3": {
			Id:        "3",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_CPU},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING},
		},
		"/agents/Node 1/slots/4": {
			Id:        "4",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_CPU},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING},
		},
		"/agents/Node 1/slots/5": {
			Id:        "5",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_CPU},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING},
		},
		"/agents/Node 1/slots/6": {
			Id:      "6",
			Device:  &devicev1.Device{Type: devicev1.Type_TYPE_CPU},
			Enabled: true,
		},
		"/agents/Node 1/slots/7": {
			Id:      "7",
			Device:  &devicev1.Device{Type: devicev1.Type_TYPE_CPU},
			Enabled: true,
		},
	}

	want1 := map[string]*agentv1.Slot{
		"/agents/Node 2/slots/0": {
			Id:        "0",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_ROCM},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING},
		},

		"/agents/Node 2/slots/1": {
			Id:      "1",
			Device:  &devicev1.Device{Type: devicev1.Type_TYPE_ROCM},
			Enabled: true,
		},
	}

	want2 := map[string]*agentv1.Slot{
		"/agents/Node 3/slots/0": {
			Id:        "0",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_CUDA},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING},
		},

		"/agents/Node 3/slots/1": {
			Id:      "1",
			Device:  &devicev1.Device{Type: devicev1.Type_TYPE_CUDA},
			Enabled: true,
		},
	}

	wantSlots := []map[string]*agentv1.Slot{want0, want1, want2}

	m.dbState.DisabledAgents = []string{"Node 2"}

	resp, err := m.GetAgents()
	require.NoError(t, err)
	assert.Equal(t, len(resp.Agents), len(nodes))

	for i, agent := range resp.Agents {
		assert.Equal(t, agent.Id, nodes[i].Name)
		assert.DeepEqual(t, agent.ResourcePools, expectedResourcePools[i])
		assert.DeepEqual(t, agent.Addresses, nodes[i].Addresses)
		assert.Equal(t, agent.Draining, nodes[i].Draining)
		assert.Equal(t, agent.Enabled, agent.Id != "Node 2")
		assert.Equal(t, len(agent.Slots), len(wantSlots[i]))
		for key, value := range agent.Slots {
			wantValue := wantSlots[i][key]
			assert.Equal(t, value.Id, wantValue.Id)
			assert.Equal(t, value.Device.Type, wantValue.Device.Type)
			assert.Equal(t, value.Enabled, wantValue.Enabled)

			if wantValue.Container != nil {
				assert.Equal(t, value.Container.State, wantValue.Container.State)
			} else if value.Container != nil {
				t.Errorf("agent.Slots %s Container value error", key)
			}
		}
	}
}

func TestHealthCheck(t *testing.T) {
	m := &DispatcherResourceManager{
		syslog: logrus.WithField("component", "dispatcherrm"),
		rmConfig: &config.DispatcherResourceManagerConfig{
			Name: "testname",
		},
	}

	c, err := newLauncherAPIClient(m.rmConfig)
	require.NoError(t, err)
	m.apiClient = c

	require.Equal(t, []model.ResourceManagerHealth{
		{
			Name:   "testname",
			Status: model.Unhealthy, // Unhealthy since launcher API client isn't set up properly.
		},
	}, m.HealthCheck())
}

func Test_summarizeResourcePool(t *testing.T) {
	type args struct {
		wlmType          wlmType
		launcherPoolName string
	}

	type want struct {
		pools         []resourcepoolv1.ResourcePool
		wlmName       string
		schedulerType resourcepoolv1.SchedulerType
		fittingPolicy resourcepoolv1.FittingPolicy
	}

	p1 := hpcPartitionDetails{
		TotalAvailableNodes:    10,
		PartitionName:          "partition 1",
		IsDefault:              true,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 2,
		TotalNodes:             10,
		TotalGpuSlots:          5,
		Accelerator:            "tesla",
	}
	p2 := hpcPartitionDetails{
		TotalAvailableNodes:    12,
		PartitionName:          "partition 2",
		IsDefault:              false,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 0,
		TotalNodes:             12,
		TotalGpuSlots:          0,
		TotalCPUSlots:          20,
		TotalAvailableCPUSlots: 8,
	}
	p3 := hpcPartitionDetails{
		TotalAvailableNodes:    15,
		PartitionName:          "partition 3",
		IsDefault:              false,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 0,
		TotalNodes:             15,
		TotalGpuSlots:          7,
	}
	p4 := hpcPartitionDetails{
		TotalAvailableNodes:    10,
		PartitionName:          "partition 4",
		IsDefault:              false,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 25,
		TotalNodes:             50,
		TotalGpuSlots:          40,
	}
	p5 := hpcPartitionDetails{
		TotalAvailableNodes:    0,
		PartitionName:          "partition 5",
		IsDefault:              false,
		TotalAllocatedNodes:    10,
		TotalAvailableGpuSlots: 25,
		TotalNodes:             50,
		TotalGpuSlots:          40,
	}
	p6 := hpcPartitionDetails{
		TotalAvailableNodes:    0,
		PartitionName:          "partition 6",
		IsDefault:              false,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 0,
		TotalNodes:             50,
		TotalGpuSlots:          0,
	}

	tests := []struct {
		name       string
		partitions []hpcPartitionDetails
		args       args
		want       want
	}{
		{
			name:       "One resource pool test",
			partitions: []hpcPartitionDetails{p1},
			args: args{
				wlmType: slurmSchedulerType,
			},
			want: want{
				pools: []resourcepoolv1.ResourcePool{
					{
						Name:           "partition 1",
						SlotType:       devicev1.Type_TYPE_CUDA,
						SlotsAvailable: 5,
						SlotsUsed:      3,
						NumAgents:      10,
					},
				},
				wlmName:       "Slurm",
				schedulerType: resourcepoolv1.SchedulerType_SCHEDULER_TYPE_SLURM,
				fittingPolicy: resourcepoolv1.FittingPolicy_FITTING_POLICY_SLURM,
			},
		},
		{
			name:       "One resource pool test, with one launcher-provided resource pool",
			partitions: []hpcPartitionDetails{p1},
			args: args{
				wlmType:          slurmSchedulerType,
				launcherPoolName: "launcher-pool",
			},
			want: want{
				pools: []resourcepoolv1.ResourcePool{
					{
						Name:           "partition 1",
						SlotType:       devicev1.Type_TYPE_CUDA,
						SlotsAvailable: 5,
						SlotsUsed:      3,
						NumAgents:      10,
					},
					{
						Name:           "launcher-pool",
						Description:    launcherPoolDescription,
						SlotType:       devicev1.Type_TYPE_CUDA,
						SlotsAvailable: 5,
						SlotsUsed:      3,
						NumAgents:      10,
					},
				},
				wlmName:       "Slurm",
				schedulerType: resourcepoolv1.SchedulerType_SCHEDULER_TYPE_SLURM,
				fittingPolicy: resourcepoolv1.FittingPolicy_FITTING_POLICY_SLURM,
			},
		},
		{
			name:       "Two resource pool test; WLM is PBS",
			partitions: []hpcPartitionDetails{p1, p2},
			args: args{
				wlmType: pbsSchedulerType,
			},
			want: want{
				pools: []resourcepoolv1.ResourcePool{
					{
						Name:           "partition 1",
						SlotType:       devicev1.Type_TYPE_CUDA,
						SlotsAvailable: 5,
						SlotsUsed:      3,
						NumAgents:      10,
						Accelerator:    "tesla",
					},
					{
						Name:           "partition 2",
						SlotType:       devicev1.Type_TYPE_CPU,
						SlotsAvailable: 20,
						SlotsUsed:      12,
						NumAgents:      12,
						SlotsPerAgent:  1,
					},
				},
				wlmName:       "PBS",
				schedulerType: resourcepoolv1.SchedulerType_SCHEDULER_TYPE_PBS,
				fittingPolicy: resourcepoolv1.FittingPolicy_FITTING_POLICY_PBS,
			},
		},
		{
			name:       "Three resource pool test",
			partitions: []hpcPartitionDetails{p1, p2, p3},
			args: args{
				wlmType: "mystery",
			},
			want: want{
				pools: []resourcepoolv1.ResourcePool{
					{
						Name:           "partition 1",
						SlotType:       devicev1.Type_TYPE_CUDA,
						SlotsAvailable: 5,
						SlotsUsed:      3,
						NumAgents:      10,
					},
					{
						Name:           "partition 2",
						SlotType:       devicev1.Type_TYPE_CPU,
						SlotsAvailable: 20,
						SlotsUsed:      12,
						NumAgents:      12,
						SlotsPerAgent:  1,
					},
					{
						Name:           "partition 3",
						SlotType:       devicev1.Type_TYPE_CUDA,
						SlotsAvailable: 7,
						SlotsUsed:      7,
						NumAgents:      15,
					},
				},
				wlmName:       "Unknown",
				schedulerType: resourcepoolv1.SchedulerType_SCHEDULER_TYPE_UNSPECIFIED,
				fittingPolicy: resourcepoolv1.FittingPolicy_FITTING_POLICY_UNSPECIFIED,
			},
		},
		{
			name:       "Available nodes with no allocated nodes",
			partitions: []hpcPartitionDetails{p4},
			args: args{
				wlmType: "mystery",
			},
			want: want{
				pools: []resourcepoolv1.ResourcePool{
					{
						Name:           "partition 4",
						SlotType:       devicev1.Type_TYPE_CUDA,
						SlotsAvailable: 40,
						SlotsUsed:      15,
						NumAgents:      10,
						SlotsPerAgent:  4,
					},
				},
				wlmName:       "Unknown",
				schedulerType: resourcepoolv1.SchedulerType_SCHEDULER_TYPE_UNSPECIFIED,
				fittingPolicy: resourcepoolv1.FittingPolicy_FITTING_POLICY_UNSPECIFIED,
			},
		},
		{
			name:       "Allocated nodes with no available nodes",
			partitions: []hpcPartitionDetails{p5},
			args: args{
				wlmType: "mystery",
			},
			want: want{
				pools: []resourcepoolv1.ResourcePool{
					{
						Name:           "partition 5",
						SlotType:       devicev1.Type_TYPE_CUDA,
						SlotsAvailable: 40,
						SlotsUsed:      15,
						NumAgents:      10,
						SlotsPerAgent:  4,
					},
				},
				wlmName:       "Unknown",
				schedulerType: resourcepoolv1.SchedulerType_SCHEDULER_TYPE_UNSPECIFIED,
				fittingPolicy: resourcepoolv1.FittingPolicy_FITTING_POLICY_UNSPECIFIED,
			},
		},
		{
			name:       "No allocated nodes and no available nodes",
			partitions: []hpcPartitionDetails{p6},
			args: args{
				wlmType: "mystery",
			},
			want: want{
				pools: []resourcepoolv1.ResourcePool{
					{
						Name:           "partition 6",
						SlotType:       devicev1.Type_TYPE_CPU,
						SlotsAvailable: 0,
						SlotsUsed:      0,
						NumAgents:      0,
						SlotsPerAgent:  0,
					},
				},
				wlmName:       "Unknown",
				schedulerType: resourcepoolv1.SchedulerType_SCHEDULER_TYPE_UNSPECIFIED,
				fittingPolicy: resourcepoolv1.FittingPolicy_FITTING_POLICY_UNSPECIFIED,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hpcResource := &hpcResources{
				Partitions: tt.partitions,
			}

			expectedName := "testnamerm"
			expectedMetadata := map[string]string{"abc": "dce"}
			overrides := make(map[string]config.DispatcherPartitionOverrideConfigs)
			rmConfig := &config.DispatcherResourceManagerConfig{
				Name:               expectedName,
				Metadata:           expectedMetadata,
				PartitionOverrides: overrides,
			}

			dpPools := []config.ResourcePoolConfig{}
			if tt.args.launcherPoolName != "" {
				hpcProvider := provconfig.HpcClusterConfig{
					Partition: tt.partitions[0].PartitionName,
				}

				dpPool1Provider := provconfig.Config{
					HPC: &hpcProvider,
				}

				dpPool1 := config.ResourcePoolConfig{
					PoolName:    tt.args.launcherPoolName,
					Description: launcherPoolDescription,
					Provider:    &dpPool1Provider,
				}
				dpPools = []config.ResourcePoolConfig{dpPool1}
			}

			m := &DispatcherResourceManager{
				wlmType:         tt.args.wlmType,
				rmConfig:        rmConfig,
				hpcDetailsCache: makeTestHpcDetailsCache(hpcResource),
				poolConfig:      dpPools,
			}

			res, _ := m.GetResourcePools()

			require.Len(t, res.ResourcePools, len(tt.want.pools))
			for i, pool := range res.ResourcePools {
				require.Equal(t, tt.want.pools[i].Name, pool.Name)
				require.Equal(t, tt.want.pools[i].SlotType, pool.SlotType)
				require.Equal(t, tt.want.pools[i].SlotsAvailable, pool.SlotsAvailable)
				require.Equal(t, tt.want.pools[i].SlotsUsed, pool.SlotsUsed)
				require.Equal(t, tt.want.pools[i].NumAgents, pool.NumAgents)
				wantDescription := tt.want.pools[i].Description
				if wantDescription == "" {
					wantDescription = tt.want.wlmName + "-managed pool of resources"
				}
				require.Equal(t, wantDescription, pool.Description)
				require.Equal(t, resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_STATIC, pool.Type)
				require.Equal(t, tt.want.pools[i].SlotsPerAgent, pool.SlotsPerAgent)
				require.Zero(t, pool.AuxContainerCapacityPerAgent)
				require.Equal(t, tt.want.schedulerType, pool.SchedulerType)
				require.Equal(t, tt.want.fittingPolicy, pool.SchedulerFittingPolicy)
				require.Empty(t, pool.Location)
				require.Empty(t, pool.InstanceType)
				require.Empty(t, pool.ImageId)
				require.Equal(t, expectedName, pool.ResourceManagerName)
				require.Equal(t, expectedMetadata, pool.ResourceManagerMetadata)
			}
		})
	}
}

func Test_dispatcherResourceManager_getPartitionValidationResponse(t *testing.T) {
	type fields struct {
		poolConfig        []config.ResourcePoolConfig
		containerDefaults *model.TaskContainerDefaultsConfig
	}
	type args struct {
		hpcDetails          hpcResources
		targetPartitionName string
	}
	type want struct {
		wantResp           hasSlurmPartitionResponse
		expectedErrorCount int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
		errors []string
	}{
		{
			name:   "resource pool not found",
			fields: fields{},
			args: args{
				hpcDetails:          hpcResources{},
				targetPartitionName: "partition-is-not-present",
			},
			want: want{wantResp: hasSlurmPartitionResponse{}},
		},
		{
			name:   "resource pool is a discovered partition",
			fields: fields{},
			args: args{
				hpcDetails: hpcResources{
					Partitions: []hpcPartitionDetails{{
						PartitionName: "target-pool",
					}},
				},
				targetPartitionName: "target-pool",
			},
			want: want{wantResp: hasSlurmPartitionResponse{
				HasResourcePool: true,
			}},
		},
		{
			name: "launcher-provided pool, but partition not present",
			fields: fields{
				poolConfig: []config.ResourcePoolConfig{{
					PoolName:    "partition-is-launcher-provided",
					Description: launcherPoolDescription,
					Provider: &provconfig.Config{
						HPC: &provconfig.HpcClusterConfig{
							Partition: "target-pool",
						},
					},
				}},
			},
			args: args{
				hpcDetails:          hpcResources{},
				targetPartitionName: "partition-is-launcher-provided",
			},
			want: want{wantResp: hasSlurmPartitionResponse{
				HasResourcePool:    false,
				ProvidingPartition: "target-pool",
			}},
		},
		{
			name: "launcher-provided pool, and providing partition is present",
			fields: fields{
				poolConfig: []config.ResourcePoolConfig{{
					PoolName:    "partition-is-launcher-provided",
					Description: launcherPoolDescription,
					Provider: &provconfig.Config{
						HPC: &provconfig.HpcClusterConfig{
							Partition: "target-pool",
						},
					},
				}},
			},
			args: args{
				hpcDetails: hpcResources{
					Partitions: []hpcPartitionDetails{{
						PartitionName: "target-pool",
					}},
				},
				targetPartitionName: "partition-is-launcher-provided",
			},
			want: want{wantResp: hasSlurmPartitionResponse{
				HasResourcePool:    true,
				ProvidingPartition: "target-pool",
			}},
		},
		{
			name: "launcher-provided pool, providing partition is present, BUT validation errors",
			fields: fields{
				poolConfig: []config.ResourcePoolConfig{{
					PoolName:    "partition-is-launcher-provided",
					Description: launcherPoolDescription,
					Provider: &provconfig.Config{
						HPC: &provconfig.HpcClusterConfig{Partition: "target-pool"},
					},
				}},
				containerDefaults: &model.TaskContainerDefaultsConfig{
					// Both these Slurm & PBS configs will contribute one error
					Slurm: expconf.SlurmConfigV0{
						RawSlotsPerNode: new(int),
						RawGpuType:      new(string),
						RawSbatchArgs:   []string{"--gpus=6"},
					},
					Pbs: expconf.PbsConfigV0{
						RawSlotsPerNode: new(int),
						RawSbatchArgs:   []string{"-c"},
					},
				},
			},
			args: args{
				hpcDetails: hpcResources{
					Partitions: []hpcPartitionDetails{{
						PartitionName: "target-pool",
					}},
				},
				targetPartitionName: "partition-is-launcher-provided",
			},
			want: want{
				wantResp: hasSlurmPartitionResponse{
					HasResourcePool:    true,
					ProvidingPartition: "target-pool",
				},
				expectedErrorCount: 2,
			},
		},
		{
			name: "launcher-provided pool, but providing partition definition absent",
			fields: fields{
				poolConfig: []config.ResourcePoolConfig{{
					PoolName:    "partition-is-launcher-provided",
					Description: launcherPoolDescription,
					Provider: &provconfig.Config{
						HPC: &provconfig.HpcClusterConfig{},
					},
				}},
			},
			args: args{
				hpcDetails: hpcResources{
					Partitions: []hpcPartitionDetails{{
						PartitionName: "target-pool",
					}},
				},
				targetPartitionName: "partition-is-launcher-provided",
			},
			want: want{wantResp: hasSlurmPartitionResponse{
				HasResourcePool: false,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load container defaults if the test specified them
			if len(tt.fields.poolConfig) > 0 && tt.fields.containerDefaults != nil {
				tt.fields.poolConfig[0].TaskContainerDefaults = tt.fields.containerDefaults
			}
			m := &DispatcherResourceManager{
				poolConfig: tt.fields.poolConfig,
			}
			resp := m.hasSlurmPartition(&tt.args.hpcDetails, tt.args.targetPartitionName)

			if resp.HasResourcePool != tt.want.wantResp.HasResourcePool {
				t.Errorf("dispatcherResourceManager.getPartitionValidationResponse() = %v, want %v",
					resp.HasResourcePool, tt.want.wantResp.HasResourcePool)
			}
			if resp.ProvidingPartition != tt.want.wantResp.ProvidingPartition {
				t.Errorf("dispatcherResourceManager.getPartitionValidationResponse() = %v, want %v",
					resp.ProvidingPartition, tt.want.wantResp.ProvidingPartition)
			}
			if len(resp.ValidationErrors) != tt.want.expectedErrorCount {
				t.Errorf("dispatcherResourceManager.getPartitionValidationResponse() = %v, want %v",
					resp.ValidationErrors, tt.want.expectedErrorCount)
			}
		})
	}
}

func makeTestHpcDetailsCache(v *hpcResources) *hpcResourceDetailsCache {
	var hpcDetailsDetails hpcResourceDetailsCache
	hpcDetailsDetails.lastSample.Store(v)
	c := make(chan struct{})
	close(c)
	hpcDetailsDetails.sampled = c
	hpcDetailsDetails.rmConfig = &config.DispatcherResourceManagerConfig{}
	return &hpcDetailsDetails
}

func Test_dispatcherResourceManager_getTaskContainerDefaults(t *testing.T) {
	// Set up some configurations with distinct values for DtrainNetworkInterface
	// so we can tell where the result was obtained from.
	partitionOverrideConfig := config.DispatcherPartitionOverrideConfigs{
		TaskContainerDefaultsConfig: &model.TaskContainerDefaultsConfig{
			DtrainNetworkInterface: "fromPartitionOverride",
		},
		Description: launcherPoolDescription,
	}
	partitionOverrides := make(map[string]config.DispatcherPartitionOverrideConfigs)
	partitionOverrides[""] = partitionOverrideConfig

	resourcePoolConfig := config.ResourcePoolConfig{
		TaskContainerDefaults: &model.TaskContainerDefaultsConfig{
			DtrainNetworkInterface: "fromResourcePoolConfig",
		},
	}

	type fields struct {
		rmConfig   *config.DispatcherResourceManagerConfig
		poolConfig []config.ResourcePoolConfig
	}
	type args struct {
		msg taskContainerDefaults
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    model.TaskContainerDefaultsConfig
		wantErr bool
	}{
		{
			name: "Use the default result",
			fields: fields{
				rmConfig: &config.DispatcherResourceManagerConfig{},
			},
			args: args{
				msg: taskContainerDefaults{},
			},
			want:    model.TaskContainerDefaultsConfig{},
			wantErr: false,
		},
		{
			name: "Use partition override",
			fields: fields{
				rmConfig: &config.DispatcherResourceManagerConfig{
					PartitionOverrides: partitionOverrides,
				},
			},
			args: args{
				msg: taskContainerDefaults{},
			},
			want: model.TaskContainerDefaultsConfig{
				DtrainNetworkInterface: "fromPartitionOverride",
			},
			wantErr: false,
		},
		{
			name: "Use pool override",
			fields: fields{
				rmConfig:   &config.DispatcherResourceManagerConfig{},
				poolConfig: []config.ResourcePoolConfig{resourcePoolConfig},
			},
			args: args{
				msg: taskContainerDefaults{},
			},
			want: model.TaskContainerDefaultsConfig{
				DtrainNetworkInterface: "fromResourcePoolConfig",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DispatcherResourceManager{
				rmConfig:   tt.fields.rmConfig,
				poolConfig: tt.fields.poolConfig,
			}
			got, err := m.TaskContainerDefaults(
				rm.ResourcePoolName(tt.args.msg.resourcePool),
				tt.args.msg.fallbackDefault,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTaskContainerDefaults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getTaskContainerDefaults() = %v, want %v", got, tt.want)
			}
		})
	}
}
