package tasks

import (
	"reflect"
	"testing"

	"github.com/determined-ai/determined/master/pkg/device"
)

const workDir = "/workdir"

func TestTaskSpec_computeLaunchConfig(t *testing.T) {
	type args struct {
		slotType       device.Type
		workDir        string
		slurmPartition string
	}
	tests := []struct {
		name string
		args args
		want *map[string]string
	}{
		{
			name: "Dispatcher is notified that CUDA support required",
			args: args{
				slotType:       device.CUDA,
				workDir:        workDir,
				slurmPartition: "partitionName",
			},
			want: &map[string]string{
				"workingDir":          workDir,
				"enableNvidia":        trueValue,
				"enableWritableTmpFs": trueValue,
				"partition":           "partitionName",
			},
		},
		{
			name: "Dispatcher is notified that ROCM support required",
			args: args{
				slotType:       device.ROCM,
				workDir:        workDir,
				slurmPartition: "partitionName",
			},
			want: &map[string]string{
				"workingDir":          workDir,
				"enableROCM":          trueValue,
				"enableWritableTmpFs": trueValue,
				"partition":           "partitionName",
			},
		},
		{
			name: "Verify behavior when no partition specified",
			args: args{
				slotType:       device.CUDA,
				workDir:        workDir,
				slurmPartition: "",
			},
			want: &map[string]string{
				"workingDir":          workDir,
				"enableNvidia":        trueValue,
				"enableWritableTmpFs": trueValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TaskSpec{}
			if got := tr.computeLaunchConfig(
				tt.args.slotType,
				tt.args.workDir,
				tt.args.slurmPartition); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TaskSpec.computeLaunchConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
