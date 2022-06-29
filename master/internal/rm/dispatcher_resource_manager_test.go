package resourcemanagers

import (
	"testing"

	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

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
