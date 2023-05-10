package podman

import (
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/cproto"
)

func Test_addHostMounts(t *testing.T) {
	type args struct {
		m    mount.Mount
		args []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Simple case",
			args: args{
				m: mount.Mount{
					Source:      "/host",
					Target:      "/container",
					BindOptions: &mount.BindOptions{},
				},
				args: []string{},
			},
			want: []string{"--volume", "/host:/container"},
		},
		{
			name: "Read-only",
			args: args{
				m: mount.Mount{
					Source:   "/host",
					Target:   "/container",
					ReadOnly: true,
				},
				args: []string{},
			},
			want: []string{"--volume", "/host:/container:ro"},
		},
		{
			name: "Read-only with propagation",
			args: args{
				m: mount.Mount{
					Type:     "",
					Source:   "/host",
					Target:   "/container",
					ReadOnly: true,
					BindOptions: &mount.BindOptions{
						Propagation: "rprivate",
					},
				},
				args: []string{},
			},
			want: []string{"--volume", "/host:/container:ro,rprivate"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostMountsToPodmanArgs(tt.args.m, tt.args.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addHostMounts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processCapabilities(t *testing.T) {
	type args struct {
		req  cproto.RunSpec
		args []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Add/drop caps test",
			args: args{
				req: cproto.RunSpec{
					HostConfig: container.HostConfig{
						CapAdd:  []string{"add-one", "add-two"},
						CapDrop: []string{"drop-two", "drop-one"},
					},
				},
				args: []string{},
			},
			want: []string{
				"--cap-add", "add-one", "--cap-add", "add-two",
				"--cap-drop", "drop-two", "--cap-drop", "drop-one",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := capabilitiesToPodmanArgs(tt.args.req, tt.args.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processCapabilities() = %v, want %v", got, tt.want)
			}
		})
	}
}
