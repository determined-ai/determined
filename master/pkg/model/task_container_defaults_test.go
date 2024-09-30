//nolint:exhaustruct
package model

import (
	"encoding/json"
	"testing"

	"github.com/docker/docker/api/types/registry"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestEnvironmentVarsDefaultMerging(t *testing.T) {
	defaultGpuType := "tesla"
	defaultSlotsPerNode := 99
	defaultPreemptionTimeout := DefaultPreemptionTimeout

	expGpuType := "a100"
	expSlurmSlotsPerNode := 8
	expPreemptionTimeout := 60

	expSlurmConfig := expconf.SlurmConfigV0{
		RawGpuType:      &expGpuType,
		RawSlotsPerNode: &expSlurmSlotsPerNode,
		RawSbatchArgs:   []string{"-SlrumExpConf"},
	}
	expPbsConfig := expconf.PbsConfigV0{
		RawSbatchArgs: []string{"-PbsExpConf"},
	}

	defaults := &TaskContainerDefaultsConfig{
		EnvironmentVariables: &RuntimeItems{
			CPU:  []string{"cpu=default"},
			CUDA: []string{"cuda=default"},
			ROCM: []string{"rocm=default"},
		},
		Slurm: expconf.SlurmConfigV0{
			RawGpuType:    &defaultGpuType,
			RawSbatchArgs: []string{"-SlrumTaskDefault"},
		},
		Pbs: expconf.PbsConfigV0{
			RawSlotsPerNode: &defaultSlotsPerNode,
			RawSbatchArgs:   []string{"-WpbsTaskDefault"},
		},
		PreemptionTimeout: defaultPreemptionTimeout,
	}
	conf := expconf.ExperimentConfig{
		RawEnvironment: &expconf.EnvironmentConfig{
			RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
				RawCPU:  []string{"cpu=expconf"},
				RawCUDA: []string{"extra=expconf"},
			},
		},
		RawSlurmConfig:       &expSlurmConfig,
		RawPbsConfig:         &expPbsConfig,
		RawPreemptionTimeout: &expPreemptionTimeout,
	}

	defaults.MergeIntoExpConfig(&conf)

	require.Equal(t, &expconf.EnvironmentVariablesMap{
		RawCPU:  []string{"cpu=default", "cpu=expconf"},
		RawCUDA: []string{"cuda=default", "extra=expconf"},
		RawROCM: []string{"rocm=default"},
	}, conf.RawEnvironment.RawEnvironmentVariables)

	require.Equal(t, expGpuType, *conf.RawSlurmConfig.RawGpuType)
	require.Equal(t, expSlurmSlotsPerNode, *conf.RawSlurmConfig.RawSlotsPerNode)
	require.Equal(t, []string{"-SlrumTaskDefault", "-SlrumExpConf"}, conf.RawSlurmConfig.SbatchArgs())
	require.Equal(t, defaultSlotsPerNode, *conf.RawPbsConfig.RawSlotsPerNode)
	require.Equal(t, []string{"-WpbsTaskDefault", "-PbsExpConf"}, conf.RawPbsConfig.SbatchArgs())
	require.Equal(t, expPreemptionTimeout, *conf.RawPreemptionTimeout)
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
		CheckpointGCPodSpec: &k8sV1.Pod{
			Spec: k8sV1.PodSpec{
				Volumes: []k8sV1.Volume{
					{
						Name: "some CHECKPOINT GC vol",
					},
				},
			},
		},
		Image: &RuntimeItem{
			CPU:  "docker://ubuntu",
			CUDA: "docker://ubuntucuda",
			ROCM: "docker://ubunturocm",
		},
		RegistryAuth: &registry.AuthConfig{
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
	remergedFilledTaskContainerDefaults := filledTaskContainerDefaults
	remergedFilledTaskContainerDefaults.Slurm.SetSbatchArgs(append(
		filledTaskContainerDefaults.Slurm.SbatchArgs(),
		filledTaskContainerDefaults.Slurm.SbatchArgs()...,
	))
	remergedFilledTaskContainerDefaults.Pbs.SetSbatchArgs(append(
		filledTaskContainerDefaults.Pbs.SbatchArgs(),
		filledTaskContainerDefaults.Pbs.SbatchArgs()...,
	))

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
				PreemptionTimeout:      60,
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
				CheckpointGCPodSpec: &k8sV1.Pod{
					Spec: k8sV1.PodSpec{
						Volumes: []k8sV1.Volume{
							{
								Name: "some CHECKPOINT GC vol 2",
							},
						},
					},
				},
				Image: &RuntimeItem{
					CPU:  "docker://ubuntu2",
					CUDA: "docker://ubuntucuda2",
					ROCM: "docker://ubunturocm2",
				},
				RegistryAuth: &registry.AuthConfig{
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
				PreemptionTimeout:      60,
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
				CheckpointGCPodSpec: &k8sV1.Pod{
					Spec: k8sV1.PodSpec{
						Volumes: []k8sV1.Volume{
							{
								Name: "some CHECKPOINT GC vol 2",
							},
						},
					},
				},
				Image: &RuntimeItem{
					CPU:  "docker://ubuntu2",
					CUDA: "docker://ubuntucuda2",
					ROCM: "docker://ubunturocm2",
				},
				RegistryAuth: &registry.AuthConfig{
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
					RawSbatchArgs:   []string{"--gpus-per-node=8", "--another2", "--gpus-per-node=6", "--another"},
				},
				Pbs: expconf.PbsConfigV0{
					RawSlotsPerNode: ptrs.Ptr(2),
					RawSbatchArgs:   []string{"--some-pbs-arg=8", "--another2", "--some-pbs-arg=5", "--another"},
				},
			},
			wantErr: false,
		},
		{
			name:    "merge other has same settings",
			self:    filledTaskContainerDefaults,
			other:   filledTaskContainerDefaults,
			want:    remergedFilledTaskContainerDefaults,
			wantErr: false,
		},
		{
			name:    "merge other has no settings",
			self:    filledTaskContainerDefaults,
			other:   TaskContainerDefaultsConfig{},
			want:    filledTaskContainerDefaults,
			wantErr: false,
		},
		{
			name: "merge startup scripts: self set",
			self: TaskContainerDefaultsConfig{
				StartupHook: "a",
			},
			other: TaskContainerDefaultsConfig{},
			want: TaskContainerDefaultsConfig{
				StartupHook: "a",
			},
			wantErr: false,
		},
		{
			name: "merge startup scripts: other set",
			self: TaskContainerDefaultsConfig{},
			other: TaskContainerDefaultsConfig{
				StartupHook: "a",
			},
			want: TaskContainerDefaultsConfig{
				StartupHook: "a",
			},
			wantErr: false,
		},
		{
			name: "merge startup scripts: both set",
			self: TaskContainerDefaultsConfig{
				StartupHook: "self",
			},
			other: TaskContainerDefaultsConfig{
				StartupHook: "other",
			},
			want: TaskContainerDefaultsConfig{
				StartupHook: "other",
			},
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

func TestLogPatternUnmarshal(t *testing.T) {
	var tcd TaskContainerDefaultsConfig
	require.NoError(t, json.Unmarshal([]byte(string(`{
		    "log_policies": [
		        {"pattern": "test", "action": {"type": "exclude_node"}},
		        {"pattern": "test2", "action": {"type": "cancel_retries"}}
		    ]
		}`)), &tcd))

	expected := TaskContainerDefaultsConfig{
		ShmSizeBytes:      4294967296,
		NetworkMode:       "bridge",
		PreemptionTimeout: DefaultPreemptionTimeout,
		LogPolicies: &expconf.LogPoliciesConfig{
			expconf.LogPolicy{RawPattern: "test", RawAction: &expconf.LogAction{
				RawExcludeNode: &expconf.LogActionExcludeNode{},
			}},
			expconf.LogPolicy{RawPattern: "test2", RawAction: &expconf.LogAction{
				RawCancelRetries: &expconf.LogActionCancelRetries{},
			}},
		},
	}
	require.Equal(t, expected, tcd)
}

func TestLogPatternPoliciesMerging(t *testing.T) {
	defaults := &TaskContainerDefaultsConfig{
		LogPolicies: &expconf.LogPoliciesConfig{
			expconf.LogPolicy{RawPattern: "a", RawAction: &expconf.LogAction{
				RawCancelRetries: &expconf.LogActionCancelRetries{},
			}},
			expconf.LogPolicy{RawPattern: "b", RawAction: &expconf.LogAction{
				RawExcludeNode: &expconf.LogActionExcludeNode{},
			}},
		},
	}

	conf := expconf.ExperimentConfig{
		RawLogPolicies: &expconf.LogPoliciesConfig{
			expconf.LogPolicy{RawPattern: "b", RawAction: &expconf.LogAction{
				RawCancelRetries: &expconf.LogActionCancelRetries{},
			}},
			expconf.LogPolicy{RawPattern: "b", RawAction: &expconf.LogAction{
				RawExcludeNode: &expconf.LogActionExcludeNode{},
			}},
			expconf.LogPolicy{RawPattern: "c", RawAction: &expconf.LogAction{
				RawExcludeNode: &expconf.LogActionExcludeNode{},
			}},
		},
	}

	defaults.MergeIntoExpConfig(&conf)

	expected := &expconf.LogPoliciesConfig{
		expconf.LogPolicy{RawPattern: "a", RawAction: &expconf.LogAction{
			RawCancelRetries: &expconf.LogActionCancelRetries{},
		}},
		expconf.LogPolicy{RawPattern: "b", RawAction: &expconf.LogAction{
			RawExcludeNode: &expconf.LogActionExcludeNode{},
		}},
		expconf.LogPolicy{RawPattern: "b", RawAction: &expconf.LogAction{
			RawCancelRetries: &expconf.LogActionCancelRetries{},
		}},
		expconf.LogPolicy{RawPattern: "c", RawAction: &expconf.LogAction{
			RawExcludeNode: &expconf.LogActionExcludeNode{},
		}},
	}

	require.Equal(t, expected, conf.RawLogPolicies)
}

func TestPodSpecsDefaultMerging(t *testing.T) {
	defaults := &TaskContainerDefaultsConfig{
		CPUPodSpec: &k8sV1.Pod{
			Spec: k8sV1.PodSpec{
				SecurityContext: &k8sV1.PodSecurityContext{
					SELinuxOptions: &k8sV1.SELinuxOptions{
						Level: "cpuLevel",
						Role:  "cpuRole",
					},
				},
			},
		},
		GPUPodSpec: &k8sV1.Pod{
			Spec: k8sV1.PodSpec{
				SecurityContext: &k8sV1.PodSecurityContext{
					SELinuxOptions: &k8sV1.SELinuxOptions{
						Level: "gpuLevel",
						Role:  "gpuRole",
					},
				},
			},
		},
	}

	for i := 0; i <= 1; i++ {
		conf := expconf.ExperimentConfig{
			RawResources: &expconf.ResourcesConfig{RawSlotsPerTrial: &i},
			RawEnvironment: &expconf.EnvironmentConfig{
				RawPodSpec: &expconf.PodSpec{
					Spec: k8sV1.PodSpec{
						SecurityContext: &k8sV1.PodSecurityContext{
							SELinuxOptions: &k8sV1.SELinuxOptions{
								Level: "expconfLevel",
							},
						},
					},
				},
			},
		}
		defaults.MergeIntoExpConfig(&conf)

		expected := &expconf.PodSpec{
			Spec: k8sV1.PodSpec{
				SecurityContext: &k8sV1.PodSecurityContext{
					SELinuxOptions: &k8sV1.SELinuxOptions{
						Level: "expconfLevel",
						Role:  []string{"cpuRole", "gpuRole"}[i],
					},
				},
			},
		}
		require.Equal(t, expected, conf.RawEnvironment.RawPodSpec)
	}
}
