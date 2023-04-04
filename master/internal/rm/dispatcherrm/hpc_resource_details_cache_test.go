package dispatcherrm

import (
	"testing"

	"github.com/determined-ai/determined/master/internal/config"
)

func Test_hpcResourceDetailsCache_selectDefaultPools(t *testing.T) {
	type fields struct {
		config *config.DispatcherResourceManagerConfig
	}
	type args struct {
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

	worf := "worf"
	data := "data"

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantCompute string
		wantAux     string
	}{
		{
			name:        "One partition test",
			fields:      fields{config: &config.DispatcherResourceManagerConfig{}},
			args:        args{hpcResourceDetails: hpc},
			wantCompute: "worf",
			wantAux:     "worf",
		},
		{
			name:        "Two partition test",
			fields:      fields{config: &config.DispatcherResourceManagerConfig{}},
			args:        args{hpcResourceDetails: hpc2},
			wantCompute: "data",
			wantAux:     "worf",
		},
		{
			name:        "Three partition test",
			fields:      fields{config: &config.DispatcherResourceManagerConfig{}},
			args:        args{hpcResourceDetails: hpc3},
			wantCompute: "data",
			wantAux:     "worf",
		},
		{
			name:        "No GPU partition test",
			fields:      fields{config: &config.DispatcherResourceManagerConfig{}},
			args:        args{hpcResourceDetails: hpc4},
			wantCompute: "picard",
			wantAux:     "picard",
		},
		{
			name: "Override default test",
			fields: fields{config: &config.DispatcherResourceManagerConfig{
				DefaultComputeResourcePool: &worf,
				DefaultAuxResourcePool:     &data,
			}},
			args:        args{hpcResourceDetails: hpc3},
			wantCompute: "worf",
			wantAux:     "data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compute, aux := selectDefaultPools(
				tt.args.hpcResourceDetails,
				tt.fields.config.DefaultComputeResourcePool,
				tt.fields.config.DefaultAuxResourcePool,
			)
			if compute != tt.wantCompute {
				t.Errorf("selectDefaultPools() compute got = %v, want %v", compute, tt.wantCompute)
			}
			if aux != tt.wantAux {
				t.Errorf("selectDefaultPools() aux got = %v, want %v", aux, tt.wantAux)
			}
		})
	}
}
