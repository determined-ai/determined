//nolint:exhaustivestruct
package model

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestEnvironmentVarsDefaultMerging(t *testing.T) {
	gpuType := "tesla"
	pbsSlotsPerNode := 99
	defaults := &TaskContainerDefaultsConfig{
		EnvironmentVariables: &RuntimeItems{
			CPU:  []string{"cpu=default"},
			CUDA: []string{"cuda=default"},
			ROCM: []string{"rocm=default"},
		},
		Slurm: expconf.SlurmConfigV0{
			RawGpuType: &gpuType,
		},
		Pbs: expconf.PbsConfigV0{
			RawSlotsPerNode: &pbsSlotsPerNode,
		},
	}
	conf := expconf.ExperimentConfig{
		RawEnvironment: &expconf.EnvironmentConfig{
			RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
				RawCPU:  []string{"cpu=expconf"},
				RawCUDA: []string{"extra=expconf"},
			},
		},
	}
	defaults.MergeIntoExpConfig(&conf)

	require.Equal(t, conf.RawEnvironment.RawEnvironmentVariables,
		&expconf.EnvironmentVariablesMap{
			RawCPU:  []string{"cpu=default", "cpu=expconf"},
			RawCUDA: []string{"cuda=default", "extra=expconf"},
			RawROCM: []string{"rocm=default"},
		})

	require.Equal(t, *conf.RawSlurmConfig.RawGpuType, gpuType)
	require.Equal(t, *conf.RawPbsConfig.RawSlotsPerNode, pbsSlotsPerNode)
}

func TestTaskContainerDefaultsConfigMerging(t *testing.T) {
	filledTaskContainerDefaults := TaskContainerDefaultsConfig{
		DtrainNetworkInterface: "ens0",
		NCCLPortRange:          "1-2",
		GLOOPortRange:          "3-4",
		ShmSizeBytes:           1234,
		NetworkMode:            "host",
		CPUPodSpec: &k8sV1.Pod{
			Spec: k8sV1.PodSpec{
				Volumes: []k8sV1.Volume{
					{
						Name: "some CPU vol",
					},
				},
			},
		},
		GPUPodSpec: &k8sV1.Pod{
			Spec: k8sV1.PodSpec{
				Volumes: []k8sV1.Volume{
					{
						Name: "some GPU vol",
					},
				},
			},
		},
		Image: &RuntimeItem{
			CPU:  "docker://ubuntu",
			CUDA: "docker://ubuntucuda",
			ROCM: "docker://ubunturocm",
		},
		RegistryAuth: &types.AuthConfig{
			Username:      "brad",
			Password:      "password",
			Auth:          "auth",
			Email:         "notmyemail@localhost",
			ServerAddress: "notmyserver@localhost",
			IdentityToken: "itoken",
			RegistryToken: "rtoken",
		},
		ForcePullImage: false,
		EnvironmentVariables: &RuntimeItems{
			CPU:  []string{"A=a"},
			CUDA: []string{"B=b"},
			ROCM: []string{"C=c"},
		},
		AddCapabilities:  []string{"CAP_AUDIT_CONTROL"},
		DropCapabilities: []string{"CAP_AUDIT_READ"},
		Devices: []DeviceConfig{{
			HostPath:      "/dev/a",
			ContainerPath: "/dev/a",
			Mode:          "mrw",
		}, {
			HostPath:      "/dev/b",
			ContainerPath: "/dev/b",
			Mode:          "mrw",
		}},
		BindMounts: []BindMount{{
			HostPath:      "/mnt/a",
			ContainerPath: "/mnt/a",
			ReadOnly:      true,
			Propagation:   "rprivate",
		}, {
			HostPath:      "/mnt/b",
			ContainerPath: "/mnt/b",
			ReadOnly:      true,
			Propagation:   "rprivate",
		}},
		WorkDir: ptrs.Ptr("/a/workdir"),
		Slurm: expconf.SlurmConfigV0{
			RawSlotsPerNode: ptrs.Ptr(1),
			RawGpuType:      ptrs.Ptr("a100:8"),
			RawSbatchArgs:   []string{"--gpus-per-node=6", "--another"},
		},
		Pbs: expconf.PbsConfigV0{
			RawSlotsPerNode: ptrs.Ptr(1),
			RawSbatchArgs:   []string{"--some-pbs-arg=5", "--another"},
		},
	}

	tests := []struct {
		name    string
		self    TaskContainerDefaultsConfig
		other   TaskContainerDefaultsConfig
		want    TaskContainerDefaultsConfig
		wantErr bool
	}{
		{
			name: "merge other has differ settings",
			self: filledTaskContainerDefaults,
			other: TaskContainerDefaultsConfig{
				DtrainNetworkInterface: "ib0",
				NCCLPortRange:          "3-4",
				GLOOPortRange:          "5-6",
				ShmSizeBytes:           6789,
				NetworkMode:            "bridge",
				CPUPodSpec: &k8sV1.Pod{
					Spec: k8sV1.PodSpec{
						Volumes: []k8sV1.Volume{
							{
								Name: "some CPU vol 2",
							},
						},
					},
				},
				GPUPodSpec: &k8sV1.Pod{
					Spec: k8sV1.PodSpec{
						Volumes: []k8sV1.Volume{
							{
								Name: "some GPU vol 2",
							},
						},
					},
				},
				Image: &RuntimeItem{
					CPU:  "docker://ubuntu2",
					CUDA: "docker://ubuntucuda2",
					ROCM: "docker://ubunturocm2",
				},
				RegistryAuth: &types.AuthConfig{
					Username:      "brad2",
					Password:      "password2",
					Auth:          "auth2",
					Email:         "notmyemail2@localhost",
					ServerAddress: "notmyserver2@localhost",
					IdentityToken: "itoken2",
					RegistryToken: "rtoken2",
				},
				ForcePullImage: true,
				EnvironmentVariables: &RuntimeItems{
					CPU:  []string{"A=1", "B=b"},
					CUDA: []string{"B=2", "C=c"},
					ROCM: []string{"C=3", "D=d"},
				},
				AddCapabilities:  []string{"CAP_AUDIT_CONTROL", "CAP_AUDIT_WRITE"},
				DropCapabilities: []string{"CAP_BPF"},
				Devices: []DeviceConfig{{
					HostPath:      "/dev/a2",
					ContainerPath: "/dev/a",
					Mode:          "mrw",
				}, {
					HostPath:      "/dev/c",
					ContainerPath: "/dev/c",
					Mode:          "mrw",
				}},
				BindMounts: []BindMount{{
					HostPath:      "/mnt/a2",
					ContainerPath: "/mnt/a",
					ReadOnly:      true,
					Propagation:   "rprivate",
				}, {
					HostPath:      "/mnt/c",
					ContainerPath: "/mnt/c",
					ReadOnly:      true,
					Propagation:   "rprivate",
				}},
				WorkDir: ptrs.Ptr("/a/workdir2"),
				Slurm: expconf.SlurmConfigV0{
					RawSlotsPerNode: ptrs.Ptr(2),
					RawGpuType:      ptrs.Ptr("a100:16"),
					RawSbatchArgs:   []string{"--gpus-per-node=8", "--another2"},
				},
				Pbs: expconf.PbsConfigV0{
					RawSlotsPerNode: ptrs.Ptr(2),
					RawSbatchArgs:   []string{"--some-pbs-arg=8", "--another2"},
				},
			},
			want: TaskContainerDefaultsConfig{
				DtrainNetworkInterface: "ib0",
				NCCLPortRange:          "3-4",
				GLOOPortRange:          "5-6",
				ShmSizeBytes:           6789,
				NetworkMode:            "bridge",
				CPUPodSpec: &k8sV1.Pod{
					Spec: k8sV1.PodSpec{
						Volumes: []k8sV1.Volume{
							{
								Name: "some CPU vol 2",
							},
						},
					},
				},
				GPUPodSpec: &k8sV1.Pod{
					Spec: k8sV1.PodSpec{
						Volumes: []k8sV1.Volume{
							{
								Name: "some GPU vol 2",
							},
						},
					},
				},
				Image: &RuntimeItem{
					CPU:  "docker://ubuntu2",
					CUDA: "docker://ubuntucuda2",
					ROCM: "docker://ubunturocm2",
				},
				RegistryAuth: &types.AuthConfig{
					Username:      "brad2",
					Password:      "password2",
					Auth:          "auth2",
					Email:         "notmyemail2@localhost",
					ServerAddress: "notmyserver2@localhost",
					IdentityToken: "itoken2",
					RegistryToken: "rtoken2",
				},
				ForcePullImage: true,
				EnvironmentVariables: &RuntimeItems{
					CPU:  []string{"A=1", "B=b"},
					CUDA: []string{"B=2", "C=c"},
					ROCM: []string{"C=3", "D=d"},
				},
				AddCapabilities:  []string{"CAP_AUDIT_CONTROL", "CAP_AUDIT_WRITE"},
				DropCapabilities: []string{"CAP_AUDIT_READ", "CAP_BPF"},
				Devices: []DeviceConfig{{
					HostPath:      "/dev/a2",
					ContainerPath: "/dev/a",
					Mode:          "mrw",
				}, {
					HostPath:      "/dev/c",
					ContainerPath: "/dev/c",
					Mode:          "mrw",
				}, {
					HostPath:      "/dev/b",
					ContainerPath: "/dev/b",
					Mode:          "mrw",
				}},
				BindMounts: []BindMount{{
					HostPath:      "/mnt/a2",
					ContainerPath: "/mnt/a",
					ReadOnly:      true,
					Propagation:   "rprivate",
				}, {
					HostPath:      "/mnt/c",
					ContainerPath: "/mnt/c",
					ReadOnly:      true,
					Propagation:   "rprivate",
				}, {
					HostPath:      "/mnt/b",
					ContainerPath: "/mnt/b",
					ReadOnly:      true,
					Propagation:   "rprivate",
				}},
				WorkDir: ptrs.Ptr("/a/workdir2"),
				Slurm: expconf.SlurmConfigV0{
					RawSlotsPerNode: ptrs.Ptr(2),
					RawGpuType:      ptrs.Ptr("a100:16"),
					RawSbatchArgs:   []string{"--gpus-per-node=8", "--another2"},
				},
				Pbs: expconf.PbsConfigV0{
					RawSlotsPerNode: ptrs.Ptr(2),
					RawSbatchArgs:   []string{"--some-pbs-arg=8", "--another2"},
				},
			},
			wantErr: false,
		}, {
			name:    "merge other has same settings",
			self:    filledTaskContainerDefaults,
			other:   filledTaskContainerDefaults,
			want:    filledTaskContainerDefaults,
			wantErr: false,
		}, {
			name:    "merge other has no settings",
			self:    filledTaskContainerDefaults,
			other:   TaskContainerDefaultsConfig{},
			want:    filledTaskContainerDefaults,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.self.Merge(tt.other)
			if (err != nil) != tt.wantErr {
				t.Errorf("TaskContainerDefaultsConfig.Merge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := deep.Equal(got, tt.want); diff != nil {
				for _, line := range diff {
					t.Errorf("got != want: %s", line)
				}
			}
		})
	}
}
