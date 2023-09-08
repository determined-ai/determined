package main

import (
	"os"
	"testing"
)

func Test_visibleGPUsFromEnvironment(t *testing.T) {
	tests := []struct {
		name           string
		cudaVisDev     string
		rocrVisDev     string
		wantVisDevices string
	}{
		{
			name:           "Nothing in environment",
			wantVisDevices: "",
		},
		{
			name:           "CUDA defined",
			cudaVisDev:     "A,B",
			wantVisDevices: "A,B",
		},
		{
			name:           "ROCR defined",
			rocrVisDev:     "1,2",
			wantVisDevices: "1,2",
		},
	}
	for _, tt := range tests {
		clearEnvironment(t)
		if tt.cudaVisDev != "" {
			if err := os.Setenv(cudaVisibleDevices, tt.cudaVisDev); err != nil {
				t.Errorf("Errors setting %s: %s", cudaVisibleDevices, err.Error())
			}
		}
		if tt.rocrVisDev != "" {
			if err := os.Setenv(rocrVisibleDevices, tt.rocrVisDev); err != nil {
				t.Errorf("Errors setting %s: %s", rocrVisibleDevices, err.Error())
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			if gotVisDevices := visibleGPUsFromEnvironment(); gotVisDevices != tt.wantVisDevices {
				t.Errorf("visibleGPUsFromEnvironment() = %v, want %v", gotVisDevices, tt.wantVisDevices)
			}
		})
	}
}

func clearEnvironment(t *testing.T) {
	if err := os.Unsetenv(cudaVisibleDevices); err != nil {
		t.Errorf("Error clearing %s: %s", cudaVisibleDevices, err.Error())
	}
	if err := os.Unsetenv(rocrVisibleDevices); err != nil {
		t.Errorf("Error clearing %s: %s", rocrVisibleDevices, err.Error())
	}
}
