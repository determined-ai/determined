package config

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDispatcherResourceManagerConfig_Validate(t *testing.T) {
	type fields struct {
		LauncherContainerRunType string
	}
	tests := []struct {
		name   string
		fields fields
		want   []error
	}{
		{
			name:   "Invalid type case",
			fields: fields{LauncherContainerRunType: "invalid-type"},
			want:   []error{fmt.Errorf("invalid launch container run type: 'invalid-type'")},
		},
		{
			name:   "singularity case",
			fields: fields{LauncherContainerRunType: "singularity"},
			want:   nil,
		},
		{
			name:   "podman case",
			fields: fields{LauncherContainerRunType: "podman"},
			want:   nil,
		},
		{
			name:   "enroot case",
			fields: fields{LauncherContainerRunType: "enroot"},
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DispatcherResourceManagerConfig{
				LauncherContainerRunType: tt.fields.LauncherContainerRunType,
			}
			if got := c.Validate(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DispatcherResourceManagerConfig.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
