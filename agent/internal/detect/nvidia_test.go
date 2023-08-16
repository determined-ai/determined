package detect

import (
	"os"
	"reflect"
	"testing"
)

func Test_parseVisibleDevices(t *testing.T) {
	tests := []struct {
		name string
		cuda string
		want []string
	}{
		{
			name: "Have CUDA",
			cuda: "1,2,3,4",
			want: []string{"1", "2", "3", "4"},
		},
		{
			name: "Don't have CUDA",
			want: nil,
		},
		{
			name: "PBS case",
			cuda: "GPU-1,GPU-2",
			want: []string{"GPU-1", "GPU-2"},
		},
	}
	for _, tt := range tests {
		if err := os.Unsetenv("CUDA_VISIBLE_DEVICES"); err != nil {
			t.Errorf("Error clearing CUDA_VISIBLE_DEVICES: %s", err.Error())
		}
		if tt.cuda != "" {
			if err := os.Setenv("CUDA_VISIBLE_DEVICES", tt.cuda); err != nil {
				t.Errorf("Errors setting CUDA_VISIBLE_DEVICES: %s", err.Error())
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			got := parseCudaVisibleDevices()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVisibleDevices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deviceAllocated(t *testing.T) {
	type args struct {
		allocatedDevices []string
		device           []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "CUDA_VISIBLE_DEVICES not defined",
			args: args{
				allocatedDevices: nil,
				device:           []string{"1", "", "GPU-1-UUID"},
			},
			want: true,
		},
		{
			name: "Slurm case, device 1, is allocated",
			args: args{
				allocatedDevices: []string{"0", "1", "3"},
				device:           []string{"1", "", "GPU-1-UUID"},
			},
			want: true,
		},
		{
			name: "Slurm case, device 2, is not allocated",
			args: args{
				allocatedDevices: []string{"0", "1", "3"},
				device:           []string{"2", "", "GPU-2-UUID"},
			},
			want: false,
		},
		{
			name: "PBS case, GPUs in CUDA_VISIBLE_DEVICES listed by UUID, device 1, is allocated",
			args: args{
				allocatedDevices: []string{"GPU-0-UUID", "GPU-1-UUID", "GPU-0-UUID"},
				device:           []string{"1", "", "GPU-1-UUID"},
			},
			want: true,
		},
		{
			name: "PBS case, GPUs in CUDA_VISIBLE_DEVICES listed by UUID, device 2, is not allocated",
			args: args{
				allocatedDevices: []string{"GPU-0-UUID", "GPU-1-UUID", "GPU-0-UUID"},
				device:           []string{"2", "", "GPU-2-UUID"},
			},
			want: false,
		},
		{
			name: "Like the previous test, with spaces in the device data",
			args: args{
				allocatedDevices: []string{"GPU-0-UUID", "GPU-1-UUID", "GPU-0-UUID"},
				device:           []string{" 2 ", "", " GPU-2-UUID "},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deviceAllocated(tt.args.allocatedDevices, tt.args.device); got != tt.want {
				t.Errorf("deviceAllocated() = %v, want %v", got, tt.want)
			}
		})
	}
}
