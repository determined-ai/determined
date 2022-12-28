package config

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestDispatcherResourceManagerConfig_Validate(t *testing.T) {
	type fields struct {
		LauncherContainerRunType string
		JobProjectSource         *string
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
		{
			name: "workspace case",
			fields: fields{
				LauncherContainerRunType: "enroot",
				JobProjectSource:         ptrs.Ptr("workspace"),
			},
			want: nil,
		},
		{
			name: "project case",
			fields: fields{
				LauncherContainerRunType: "enroot",
				JobProjectSource:         ptrs.Ptr("project"),
			},
			want: nil,
		},
		{
			name: "label case",
			fields: fields{
				LauncherContainerRunType: "enroot",
				JobProjectSource:         ptrs.Ptr("label"),
			},
			want: nil,
		},
		{
			name: "label: case",
			fields: fields{
				LauncherContainerRunType: "enroot",
				JobProjectSource:         ptrs.Ptr("label:something"),
			},
			want: nil,
		},
		{
			name: "invalid project source",
			fields: fields{
				LauncherContainerRunType: "enroot",
				JobProjectSource:         ptrs.Ptr("something-bad"),
			},
			want: []error{fmt.Errorf(
				"invalid job_project_source value: 'something-bad'. " +
					"Specify one of project, workspace or label[:value]")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DispatcherResourceManagerConfig{
				LauncherContainerRunType: tt.fields.LauncherContainerRunType,
				JobProjectSource:         tt.fields.JobProjectSource,
			}
			if got := c.Validate(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DispatcherResourceManagerConfig.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
