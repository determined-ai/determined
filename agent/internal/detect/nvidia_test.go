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
	}
	for _, tt := range tests {
		if err := os.Unsetenv("CUDA_VISIBLE_DEVICES"); err != nil {
			t.Errorf("Error unsetting CUDA_VISIBLE_DEVICES: %s", err.Error())
		}
		if tt.cuda != "" {
			if err := os.Setenv("CUDA_VISIBLE_DEVICES", tt.cuda); err != nil {
				t.Errorf("Errors setting CUDA_VISIBLE_DEVICES: %s", err.Error())
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			got := parseVisibleDevices()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVisibleDevices() = %v, want %v", got, tt.want)
			}
			deviceNotAllocated(got, []string{"0"})
		})
	}
}
