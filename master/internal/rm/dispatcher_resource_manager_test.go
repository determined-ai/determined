package resourcemanagers

import (
	"testing"
	"time"

	"gotest.tools/assert"

	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

func Test_authContext(t *testing.T) {
	m := &dispatcherResourceManager{
		authToken: "xyz",
	}
	ctx := &actor.Context{}
	ctxWith := m.authContext(ctx)
	if authToken := ctxWith.Value(launcher.ContextAccessToken); authToken != nil {
		assert.Equal(t, authToken, "xyz")
		return
	}
	t.Errorf("authContext failed")
}

func Test_generateGetAgentsResponse(t *testing.T) {
	n1 := hpcNodeDetails{
		Partitions:    []string{"Partition 1", "Partition 2"},
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
		Partitions:    []string{"Partition 1", "Partition 3"},
		Addresses:     []string{"address 3", "address 4"},
		Draining:      false,
		Allocated:     true,
		Name:          "Node 2",
		GpuCount:      2,
		GpuInUseCount: 1,
		CPUCount:      8,
		CPUInUseCount: 0,
	}

	nodes := []hpcNodeDetails{n1, n2}

	hpcResource := &hpcResources{
		Nodes: nodes,
	}

	resourceDetails := &hpcResourceDetailsCache{
		lastSample: *hpcResource,
		sampleTime: time.Now(),
	}

	overrides := make(map[string]config.DispatcherPartitionOverrideConfigs)
	config := &config.DispatcherResourceManagerConfig{
		PartitionOverrides: overrides,
	}

	m := &dispatcherResourceManager{
		config:          config,
		resourceDetails: *resourceDetails,
	}

	var want0 map[string]*agentv1.Slot = map[string]*agentv1.Slot{
		"/agents/Node 1/slots/8": {
			Id:        "8",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_CPU},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING}},
	}

	var want1 map[string]*agentv1.Slot = map[string]*agentv1.Slot{
		"/agents/Node 2/slots/0": {
			Id:        "0",
			Device:    &devicev1.Device{Type: devicev1.Type_TYPE_CUDA},
			Enabled:   true,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING}},

		"/agents/Node 2/slots/1": {
			Id:      "1",
			Device:  &devicev1.Device{Type: devicev1.Type_TYPE_CUDA},
			Enabled: true},
	}

	wantSlots := []map[string]*agentv1.Slot{want0, want1}

	ctx := &actor.Context{}
	resp := m.generateGetAgentsResponse(ctx)
	assert.Equal(t, len(resp.Agents), len(nodes))

	for i, agent := range resp.Agents {
		assert.Equal(t, agent.Id, nodes[i].Name)
		assert.DeepEqual(t, agent.ResourcePools, nodes[i].Partitions)
		assert.DeepEqual(t, agent.Addresses, nodes[i].Addresses)
		assert.Equal(t, agent.Draining, nodes[i].Draining)

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

func Test_summarizeResourcePool(t *testing.T) {
	type args struct {
		ctx *actor.Context
	}

	p1 := hpcPartitionDetails{
		TotalAvailableNodes:    0,
		PartitionName:          "partition 1",
		IsDefault:              true,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 2,
		TotalNodes:             10,
		TotalGpuSlots:          5,
	}
	p2 := hpcPartitionDetails{
		TotalAvailableNodes:    0,
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
		TotalAvailableNodes:    0,
		PartitionName:          "partition 3",
		IsDefault:              false,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 0,
		TotalNodes:             15,
		TotalGpuSlots:          7,
	}

	tests := []struct {
		name       string
		partitions []hpcPartitionDetails
		args       args
		want       []resourcepoolv1.ResourcePool
	}{
		{
			name:       "One resource pool test",
			partitions: []hpcPartitionDetails{p1},
			args:       args{},
			want: []resourcepoolv1.ResourcePool{
				{Name: "partition 1",
					SlotType:       devicev1.Type_TYPE_CUDA,
					SlotsAvailable: 5,
					SlotsUsed:      3,
					NumAgents:      10},
			},
		},
		{
			name:       "Two resource pool test",
			partitions: []hpcPartitionDetails{p1, p2},
			args:       args{},
			want: []resourcepoolv1.ResourcePool{
				{Name: "partition 1",
					SlotType:       devicev1.Type_TYPE_CUDA,
					SlotsAvailable: 5,
					SlotsUsed:      3,
					NumAgents:      10},

				{Name: "partition 2",
					SlotType:       devicev1.Type_TYPE_CPU,
					SlotsAvailable: 20,
					SlotsUsed:      12,
					NumAgents:      12},
			},
		},
		{
			name:       "Three resource pool test",
			partitions: []hpcPartitionDetails{p1, p2, p3},
			args:       args{},
			want: []resourcepoolv1.ResourcePool{
				{Name: "partition 1",
					SlotType:       devicev1.Type_TYPE_CUDA,
					SlotsAvailable: 5,
					SlotsUsed:      3,
					NumAgents:      10},
				{Name: "partition 2",
					SlotType:       devicev1.Type_TYPE_CPU,
					SlotsAvailable: 20,
					SlotsUsed:      12,
					NumAgents:      12},
				{Name: "partition 3",
					SlotType:       devicev1.Type_TYPE_CUDA,
					SlotsAvailable: 7,
					SlotsUsed:      7,
					NumAgents:      15},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hpcResource := &hpcResources{
				Partitions: tt.partitions,
			}

			resourceDetails := &hpcResourceDetailsCache{
				lastSample: *hpcResource,
				sampleTime: time.Now(),
			}

			overrides := make(map[string]config.DispatcherPartitionOverrideConfigs)
			config := &config.DispatcherResourceManagerConfig{
				PartitionOverrides: overrides,
			}

			m := &dispatcherResourceManager{
				config:          config,
				resourceDetails: *resourceDetails,
			}

			res, _ := m.summarizeResourcePool(tt.args.ctx)

			assert.Equal(t, len(tt.want), len(res))
			for i, pool := range res {
				assert.Equal(t, pool.Name, tt.want[i].Name)
				assert.Equal(t, pool.SlotType, tt.want[i].SlotType)
				assert.Equal(t, pool.SlotsAvailable, tt.want[i].SlotsAvailable)
				assert.Equal(t, pool.SlotsUsed, tt.want[i].SlotsUsed)
				assert.Equal(t, pool.NumAgents, tt.want[i].NumAgents)

				assert.Equal(t, pool.Description, "Slurm-managed pool of resources")
				assert.Equal(t, pool.Type, resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_STATIC)
				assert.Equal(t, pool.SlotsPerAgent, int32(0))
				assert.Equal(t, pool.AuxContainerCapacityPerAgent, int32(0))
				assert.Equal(t, pool.SchedulerType, resourcepoolv1.SchedulerType_SCHEDULER_TYPE_SLURM)
				assert.Equal(t, pool.SchedulerFittingPolicy, resourcepoolv1.FittingPolicy_FITTING_POLICY_SLURM)
				assert.Equal(t, pool.Location, "Slurm")
				assert.Equal(t, pool.InstanceType, "Slurm")
				assert.Equal(t, pool.ImageId, "")
			}
		})
	}
}

func Test_dispatcherResourceManager_selectDefaultPools(t *testing.T) {
	type fields struct {
		config                      *config.DispatcherResourceManagerConfig
		apiClient                   *launcher.APIClient
		hpcResourcesManifest        *launcher.Manifest
		reqList                     *taskList
		groups                      map[*actor.Ref]*group
		addrToResourcesID           map[*actor.Ref]sproto.ResourcesID
		resourcesIDToAddr           map[sproto.ResourcesID]*actor.Ref
		slotsUsedPerGroup           map[*group]int
		dispatchIDToAllocationID    map[string]model.AllocationID
		masterTLSConfig             model.TLSClientConfig
		loggingConfig               model.LoggingConfig
		jobWatcher                  *launcherMonitor
		authToken                   string
		resourceDetails             hpcResourceDetailsCache
		DefaultComputePoolPartition string
		DefaultAuxPoolPartition     string
	}
	type args struct {
		ctx                *actor.Context
		hpcResourceDetails []hpcPartitionDetails
	}

	p1 := hpcPartitionDetails{
		TotalAvailableNodes:    0,
		PartitionName:          "worf",
		IsDefault:              true,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 0,
		TotalNodes:             0,
		TotalGpuSlots:          0,
	}
	p2 := hpcPartitionDetails{
		TotalAvailableNodes:    0,
		PartitionName:          "data",
		IsDefault:              false,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 0,
		TotalNodes:             0,
		TotalGpuSlots:          1,
	}
	p3 := hpcPartitionDetails{
		TotalAvailableNodes:    0,
		PartitionName:          "picard",
		IsDefault:              false,
		TotalAllocatedNodes:    0,
		TotalAvailableGpuSlots: 0,
		TotalNodes:             0,
		TotalGpuSlots:          0,
	}
	hpc := []hpcPartitionDetails{
		p1,
	}
	hpc2 := []hpcPartitionDetails{
		p1, p2,
	}
	hpc3 := []hpcPartitionDetails{
		p1, p2, p3,
	}
	// One partition, no GPUs
	hpc4 := []hpcPartitionDetails{
		p3,
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantCompute string
		wantAux     string
	}{
		{
			name:        "One partition test",
			fields:      fields{},
			args:        args{hpcResourceDetails: hpc},
			wantCompute: "worf",
			wantAux:     "worf",
		},
		{
			name:        "Two partition test",
			fields:      fields{},
			args:        args{hpcResourceDetails: hpc2},
			wantCompute: "data",
			wantAux:     "worf",
		},
		{
			name:        "Three partition test",
			fields:      fields{},
			args:        args{hpcResourceDetails: hpc3},
			wantCompute: "data",
			wantAux:     "worf",
		},
		{
			name:        "No GPU partition test",
			fields:      fields{},
			args:        args{hpcResourceDetails: hpc4},
			wantCompute: "picard",
			wantAux:     "picard",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &dispatcherResourceManager{
				config:                      tt.fields.config,
				apiClient:                   tt.fields.apiClient,
				hpcResourcesManifest:        tt.fields.hpcResourcesManifest,
				reqList:                     tt.fields.reqList,
				groups:                      tt.fields.groups,
				addrToResourcesID:           tt.fields.addrToResourcesID,
				resourcesIDtoAddr:           tt.fields.resourcesIDToAddr,
				slotsUsedPerGroup:           tt.fields.slotsUsedPerGroup,
				dispatchIDToAllocationID:    tt.fields.dispatchIDToAllocationID,
				masterTLSConfig:             tt.fields.masterTLSConfig,
				loggingConfig:               tt.fields.loggingConfig,
				jobWatcher:                  tt.fields.jobWatcher,
				authToken:                   tt.fields.authToken,
				resourceDetails:             tt.fields.resourceDetails,
				defaultComputePoolPartition: tt.fields.DefaultComputePoolPartition,
				defaultAuxPoolPartition:     tt.fields.DefaultAuxPoolPartition,
			}
			compute, aux := m.selectDefaultPools(tt.args.ctx, tt.args.hpcResourceDetails)
			if compute != tt.wantCompute {
				t.Errorf("selectDefaultPools() compute got = %v, want %v", compute, tt.wantCompute)
			}
			if aux != tt.wantAux {
				t.Errorf("selectDefaultPools() aux got = %v, want %v", aux, tt.wantAux)
			}
		})
	}
}
