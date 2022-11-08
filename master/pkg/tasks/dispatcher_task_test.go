package tasks

import (
	"archive/tar"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	workDir           = "/workdir"
	pbsSlotsPerNode   = 2
	slurmSlotsPerNode = 3
)

var aug = &model.AgentUserGroup{
	ID:     1,
	UserID: 1,
	User:   "determined",
	UID:    0,
	Group:  "test-group",
	GID:    0,
}

func Test_getPortMappings(t *testing.T) {
	type args struct {
		ports map[string]int
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Verify when ports are specified",
			args: args{
				ports: map[string]int{"Podman_1": 8080, "podmanPortMapping2": 443, "PodMan3": 3000},
			},
			want: []string{
				"8080", "443", "3000",
			},
		},
		{
			name: "Verify trial, tensorboard, shell, notebook ports are ignored",
			args: args{
				ports: map[string]int{
					"PODMAN_1": 8080, "trial": 1734, "podmanPortMapping2": 443,
					"podman3": 3000, "shell": 3299, "notebook": 2988, "tensorboard": 2799,
				},
			},
			want: []string{
				"8080", "443", "3000",
			},
		},
		{
			name: "Verify portman ports are not specified",
			args: args{
				ports: map[string]int{"trial": 1734, "shell": 3450, "notPodmanPrefix": 5555},
			},
			want: []string{},
		},
		{
			name: "Verify no ports are specified",
			args: args{
				ports: map[string]int{},
			},
			want: []string{},
		},
	}
	disableImageCache := true
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment := expconf.EnvironmentConfigV0{
				RawImage:                &expconf.EnvironmentImageMapV0{},
				RawEnvironmentVariables: &expconf.EnvironmentVariablesMapV0{},
				RawRegistryAuth:         &types.AuthConfig{},
				RawForcePullImage:       &disableImageCache,
				RawPodSpec:              &expconf.PodSpec{},
				RawAddCapabilities:      []string{},
				RawDropCapabilities:     []string{},
				RawPorts:                tt.args.ports,
			}

			tr := &TaskSpec{
				Environment: environment,
			}

			got := getPortMappings(tr)

			assert.Equal(t, len(*got), len(tt.want))
			if len(tt.want) != 0 {
				gotCopy := make([]string, len(tt.want))
				wantCopy := make([]string, len(tt.want))
				copy(gotCopy, *got)
				copy(wantCopy, tt.want)
				sort.Strings(gotCopy)
				sort.Strings(wantCopy)

				if !reflect.DeepEqual(gotCopy, wantCopy) {
					t.Errorf("getPortMappings() = %v, want %v", *got, tt.want)
				}
			}
		})
	}
}

func TestTaskSpec_computeLaunchConfig(t *testing.T) {
	type args struct {
		slotType          device.Type
		workDir           string
		slurmPartition    string
		containerRunType  string
		disableImageCache bool
		addCaps           []string
		dropCaps          []string
		hostPaths         []string
		containerPaths    []string
		modes             []string
		launchingUser     string
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
				"exportAll":           trueValue,
				"queue":               "partitionName",
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
				"exportAll":           trueValue,
				"queue":               "partitionName",
			},
		},
		{
			name: "Verify behavior when no partition specified, but using podman with a specified user",
			args: args{
				slotType:         device.CUDA,
				workDir:          workDir,
				slurmPartition:   "",
				containerRunType: "podman",
				launchingUser:    "ted",
			},
			want: &map[string]string{
				"workingDir":          workDir,
				"enableNvidia":        trueValue,
				"enableWritableTmpFs": trueValue,
				"exportAll":           trueValue,
				"networkMode":         "host",
			},
		},
		{
			name: "Verify behavior when image cache disabled & capabilities are being manipulated",
			args: args{
				slotType:          device.CUDA,
				workDir:           workDir,
				slurmPartition:    "",
				containerRunType:  "podman",
				disableImageCache: true,
				addCaps:           []string{"add1", "add2"},
				dropCaps:          []string{"drop1", "drop2"},
			},
			want: &map[string]string{
				"workingDir":          workDir,
				"enableNvidia":        trueValue,
				"enableWritableTmpFs": trueValue,
				"exportAll":           trueValue,
				"networkMode":         "host",
				"disableImageCache":   trueValue,
				"addCapabilities":     "add1,add2",
				"dropCapabilities":    "drop1,drop2",
			},
		},
		{
			name: "Verify behavior when devices are specified",
			args: args{
				workDir:          workDir,
				containerRunType: "podman",
				hostPaths:        []string{"/dev/one", "/dev/two"},
				containerPaths:   []string{"/dev/tr/1", "/dev/ctr/2"},
				modes:            []string{"", "abc"},
			},
			want: &map[string]string{
				"workingDir":          workDir,
				"enableWritableTmpFs": trueValue,
				"exportAll":           trueValue,
				"networkMode":         "host",
				"devices":             "/dev/one:/dev/tr/1:,/dev/two:/dev/ctr/2:abc",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment := expconf.EnvironmentConfigV0{
				RawImage:                &expconf.EnvironmentImageMapV0{},
				RawEnvironmentVariables: &expconf.EnvironmentVariablesMapV0{},
				RawPorts:                map[string]int{},
				RawRegistryAuth:         &types.AuthConfig{},
				RawForcePullImage:       &tt.args.disableImageCache,
				RawPodSpec:              &expconf.PodSpec{},
				RawAddCapabilities:      tt.args.addCaps,
				RawDropCapabilities:     tt.args.dropCaps,
			}

			tr := &TaskSpec{
				Environment: environment,
			}

			if len(tt.args.hostPaths) > 0 {
				devices := []expconf.DeviceV0{}
				for i, hostPath := range tt.args.hostPaths {
					device := expconf.DeviceV0{
						RawHostPath:      hostPath,
						RawContainerPath: tt.args.containerPaths[i],
						RawMode:          &tt.args.modes[i],
					}
					devices = append(devices, device)
				}
				tr.ResourcesConfig = expconf.ResourcesConfigV0{
					RawSlots:          new(int),
					RawMaxSlots:       new(int),
					RawSlotsPerTrial:  new(int),
					RawWeight:         new(float64),
					RawNativeParallel: new(bool),
					RawShmSize:        new(int),
					RawAgentLabel:     new(string),
					RawResourcePool:   new(string),
					RawPriority:       new(int),
					RawDevices:        devices,
				}
			}

			if got := tr.computeLaunchConfig(
				tt.args.slotType,
				tt.args.workDir,
				tt.args.slurmPartition,
				tt.args.containerRunType,
				tt.args.launchingUser,
			); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TaskSpec.computeLaunchConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dispatcherArchive(t *testing.T) {
	err := etc.SetRootPath("../../static/srv/")
	assert.NilError(t, err)

	want := cproto.RunArchive{
		Path: "/",
		Archive: archive.Archive{
			archive.Item{
				Path:     "/determined_local_fs/dispatcher-wrapper.sh",
				Type:     tar.TypeReg,
				FileMode: os.FileMode(0o700),
				UserID:   0,
				GroupID:  0,
			},
			archive.Item{
				Path:     "/run/determined",
				Type:     tar.TypeDir,
				FileMode: os.FileMode(0o700),
				UserID:   0,
				GroupID:  0,
			},
			archive.Item{
				Path:     "/run/determined/link-1",
				Type:     tar.TypeSymlink,
				FileMode: os.FileMode(0o700),
				UserID:   0,
				GroupID:  0,
			},
			archive.Item{
				Path:     "/run/determined/link-2",
				Type:     tar.TypeSymlink,
				FileMode: os.FileMode(0o700),
				UserID:   0,
				GroupID:  0,
			},
			archive.Item{
				Path:     "/run/determined/link-3",
				Type:     tar.TypeSymlink,
				FileMode: os.FileMode(0o700),
				UserID:   0,
				GroupID:  0,
			},
		},
	}

	got := dispatcherArchive(aug, []string{"link-1", "link-2", "link-3"})

	assert.Equal(t, got.Path, want.Path)
	assert.Equal(t, len(got.Archive), len(want.Archive))
	for i, a := range got.Archive {
		assert.Equal(t, a.Path, want.Archive[i].Path)
		assert.Equal(t, a.UserID, want.Archive[i].UserID)
		assert.Equal(t, a.GroupID, want.Archive[i].GroupID)
		assert.Equal(t, a.Type, want.Archive[i].Type)
		assert.Equal(t, a.FileMode, want.Archive[i].FileMode)
	}
}

func Test_makeLocalVolumn(t *testing.T) {
	tests := []struct {
		name string
		arg  cproto.RunArchive
		want bool
	}{
		{
			name: "Test sshDir",
			arg: cproto.RunArchive{
				Archive: archive.Archive{
					archive.Item{
						Path: "/run/determined/ssh",
					},
				},
			},
			want: false,
		},
		{
			name: "Test TaskLoggingSetupScriptResource",
			arg: cproto.RunArchive{
				Archive: archive.Archive{
					archive.Item{
						Path: "task-logging-setup.sh",
					},
				},
			},
			want: false,
		},
		{
			name: "Test CommandEntryPointResource",
			arg: cproto.RunArchive{
				Archive: archive.Archive{
					archive.Item{
						Path: "/run/determined/command-entrypoint.sh",
					},
				},
			},
			want: false,
		},
		{
			name: "Test ShellEntryPointResource",
			arg: cproto.RunArchive{
				Archive: archive.Archive{
					archive.Item{
						Path: "/run/determined/shell-entrypoint.sh",
					},
				},
			},
			want: false,
		},
		{
			name: "Test item path runDir",
			arg: cproto.RunArchive{
				Archive: archive.Archive{
					archive.Item{
						Path: "/run/determined/",
					},
				},
			},
			want: true,
		},
		{
			name: "Test path runDir",
			arg: cproto.RunArchive{
				Path: "/run/determined",
			},
			want: true,
		},
		{
			name: "Test path DefaultWorkDir",
			arg: cproto.RunArchive{
				Path: "/run/determined/workdir",
			},
			want: true,
		},
		{
			name: "Test no path specified",
			arg:  cproto.RunArchive{},
			want: false,
		},
		{
			name: "Test path random",
			arg: cproto.RunArchive{
				Path: "/x/y/z",
			},
			want: false,
		},
		{
			name: "Test item path random",
			arg: cproto.RunArchive{
				Archive: archive.Archive{
					archive.Item{
						Path: "/x/y/z",
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeLocalVolume(tt.arg)
			assert.Equal(t, got, tt.want)
		})
	}
}

func Test_getAllArchives(t *testing.T) {
	ts := &TaskSpec{
		AgentUserGroup: aug,
		WorkDir:        "/run/determined/workdir",
	}

	got := getAllArchives(ts)
	assert.Assert(t, len(*got) > 0)
}

func Test_encodeArchiveParameters(t *testing.T) {
	dispatcherArchive := cproto.RunArchive{}
	allArchives := []cproto.RunArchive{
		{
			Path: "/run/determined/workdir",
		},
	}
	customArgs, err := encodeArchiveParameters(dispatcherArchive, allArchives)
	assert.NilError(t, err)
	assert.Assert(t, len(customArgs["Archives"]) > 0)
}

func Test_ToDispatcherManifest(t *testing.T) {
	err := etc.SetRootPath("../../static/srv/")
	assert.NilError(t, err)
	tests := []struct {
		name                   string
		containerRunType       string
		isPbsScheduler         bool
		slotType               device.Type
		tresSupported          bool
		gresSupported          bool
		Slurm                  []string
		Pbs                    []string
		wantCarrier            string
		wantResourcesInstances *map[string]int32
		wantResourcesGpus      *map[string]int32
		wantSlurmArgs          []string
		wantPbsArgs            []string
		wantErr                bool
		errorContains          string
	}{
		{
			name:             "Test singularity with Slurm",
			containerRunType: "singularity",
			slotType:         device.CUDA,
			tresSupported:    true,
			wantCarrier:      "com.cray.analytics.capsules.carriers.hpc.slurm.SingularityOverSlurm",
		},
		{
			name:             "Test singularity with PBS",
			containerRunType: "singularity",
			slotType:         device.CUDA,
			tresSupported:    true,
			isPbsScheduler:   true,
			wantCarrier:      "com.cray.analytics.capsules.carriers.hpc.pbs.SingularityOverPbs",
		},
		{
			name:             "Test podman with Slurm",
			containerRunType: "podman",
			slotType:         device.CPU,
			tresSupported:    false,
			wantCarrier:      "com.cray.analytics.capsules.carriers.hpc.slurm.PodmanOverSlurm",
		},
		{
			name:             "Test podman with PBS",
			containerRunType: "podman",
			slotType:         device.CPU,
			tresSupported:    false,
			isPbsScheduler:   true,
			wantCarrier:      "com.cray.analytics.capsules.carriers.hpc.pbs.PodmanOverPbs",
		},
		{
			name:             "Test TresSupported true",
			containerRunType: "singularity",
			slotType:         device.CUDA,
			tresSupported:    true,
			gresSupported:    true,
			wantCarrier:      "com.cray.analytics.capsules.carriers.hpc.slurm.SingularityOverSlurm",
			wantResourcesInstances: &map[string]int32{
				"per-node": 1,
			},
			wantResourcesGpus: &map[string]int32{
				"total": 16,
			},
		},
		{
			name:             "Test TresSupported false",
			containerRunType: "singularity",
			slotType:         device.CUDA,
			tresSupported:    false,
			gresSupported:    true,
			Slurm:            []string{},
			wantCarrier:      "com.cray.analytics.capsules.carriers.hpc.slurm.SingularityOverSlurm",
			wantResourcesInstances: &map[string]int32{
				"nodes": 16,
			},
			wantResourcesGpus: &map[string]int32{
				"per-node": 1,
			},
		},
		{
			name:             "Test GresSupported false",
			containerRunType: "singularity",
			slotType:         device.CUDA,
			tresSupported:    false,
			gresSupported:    false,
			wantResourcesInstances: &map[string]int32{
				"nodes": 16,
			},
		},
		{
			name:             "Test custom slurmArgs",
			containerRunType: "singularity",
			slotType:         device.CUDA,
			Slurm:            []string{"--want=slurmArgs", "--X=Y"},
			wantSlurmArgs:    []string{"--want=slurmArgs", "--X=Y"},
		},
		{
			name:             "Test custom pbsArgs",
			containerRunType: "singularity",
			slotType:         device.CUDA,
			Pbs:              []string{"-want=pbsArgs", "-AB"},
			wantPbsArgs:      []string{"-want=pbsArgs", "-AB"},
		},
		{
			name:             "Test invalid Slurm options",
			containerRunType: "singularity",
			slotType:         device.CUDA,
			Slurm:            []string{"--gpus=2"},
			Pbs:              []string{},
			wantErr:          true,
			errorContains:    "is not configurable",
		},
		{
			name:             "Test invalid PBS options",
			containerRunType: "singularity",
			slotType:         device.CUDA,
			Pbs:              []string{"-I"},
			wantErr:          true,
			errorContains:    "is not configurable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disableImageCache := true
			image := "RawImage"

			environment := expconf.EnvironmentConfigV0{
				RawImage: &expconf.EnvironmentImageMapV0{
					RawCPU:  &image,
					RawCUDA: &image,
					RawROCM: &image,
				},
				RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{},
				RawRegistryAuth:         &types.AuthConfig{},
				RawForcePullImage:       &disableImageCache,
				RawPodSpec:              &expconf.PodSpec{},
				RawAddCapabilities:      []string{},
				RawDropCapabilities:     []string{},
				RawPorts: map[string]int{
					"Podman_1":           8080,
					"podmanPortMapping2": 443, "PodMan3": 3000,
				},
			}
			slurmOpts := expconf.SlurmConfig{
				RawSlotsPerNode: nil,
				RawGpuType:      nil,
				RawSbatchArgs:   tt.Slurm,
			}
			pbsOpts := expconf.PbsConfig{
				RawSlotsPerNode: nil,
				RawSbatchArgs:   tt.Pbs,
			}

			ts := &TaskSpec{
				AgentUserGroup: aug,
				WorkDir:        "/run/determined/workdir",
				Environment:    environment,
				PbsConfig:      pbsOpts,
				SlurmConfig:    slurmOpts,
			}

			manifest, userName, payloadName, err := ts.ToDispatcherManifest(
				"masterHost", 8888, "certName", 16, tt.slotType,
				"slurm_partition1", tt.tresSupported, tt.gresSupported, tt.containerRunType,
				tt.isPbsScheduler)

			if tt.wantErr {
				assert.ErrorContains(t, err, tt.errorContains)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, userName, "determined")
				assert.Equal(t, payloadName, "ai")
				assert.Assert(t, manifest != nil)
				assert.Equal(t, len(*manifest.Payloads), 1)

				payload := (*manifest.Payloads)[0]
				assert.Equal(t, *payload.Name, "ai")
				assert.Equal(t, *payload.Id, "com.cray.analytics.capsules.generic.container")
				assert.Equal(t, *payload.Version, "latest")

				assert.Equal(t, len(*payload.Carriers), 1)
				if len(tt.wantCarrier) > 0 {
					assert.Equal(t, (*payload.Carriers)[0], tt.wantCarrier)
				}

				launchParameters := payload.LaunchParameters
				assert.Assert(t, launchParameters != nil)

				customs := launchParameters.GetCustom()
				assert.Assert(t, customs != nil)
				assert.Assert(t, customs["Archives"] != nil)

				if len(tt.wantSlurmArgs) > 0 {
					assert.DeepEqual(t, customs["slurmArgs"], tt.wantSlurmArgs)
				} else {
					assert.Assert(t, customs["slurmArgs"] == nil)
				}

				if len(tt.wantPbsArgs) > 0 {
					assert.DeepEqual(t, customs["pbsArgs"], tt.wantPbsArgs)
				} else {
					assert.Assert(t, customs["pbsArgs"] == nil)
				}

				if tt.containerRunType == "podman" {
					assert.Assert(t, customs["ports"] != nil)
				} else {
					assert.Assert(t, customs["ports"] == nil)
				}

				resourceRequirements := payload.ResourceRequirements
				instances := resourceRequirements.Instances
				gpus := resourceRequirements.Gpus

				if tt.wantResourcesInstances != nil {
					assert.Assert(t, instances != nil)
					assert.Assert(t, len(*instances) == len(*tt.wantResourcesInstances))
					for k, v := range *instances {
						assert.Assert(t, v == (*tt.wantResourcesInstances)[k])
					}
				}

				if tt.wantResourcesGpus != nil {
					assert.Assert(t, gpus != nil)
					assert.Assert(t, len(*gpus) == len(*tt.wantResourcesGpus))
					for k, v := range *gpus {
						assert.Assert(t, v == (*tt.wantResourcesGpus)[k])
					}
				}

				if tt.gresSupported == false {
					assert.Assert(t, gpus == nil)
				}
			}
		})
	}
}

func Test_getEnvVarsForLauncherManifest(t *testing.T) {
	disableImageCache := true

	environment := expconf.EnvironmentConfigV0{
		RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
			RawCPU:  []string{"cpu=default", "myenv=xyz"},
			RawCUDA: []string{"cuda=default", "extra=expconf"},
			RawROCM: []string{"rocm=default"},
		},
		RawRegistryAuth: &types.AuthConfig{
			Username:      "user",
			Password:      "pwd",
			ServerAddress: "addr",
			Email:         "email",
		},
		RawForcePullImage:   &disableImageCache,
		RawAddCapabilities:  []string{"add1", "add2"},
		RawDropCapabilities: []string{"drop1", "drop2"},
		RawImage:            &expconf.EnvironmentImageMapV0{},
		RawPodSpec:          &expconf.PodSpec{},
		RawPorts:            map[string]int{},
	}

	ts := &TaskSpec{
		ContainerID: "Container_ID",
		ClusterID:   "Cluster_ID",
		Environment: environment,
		Devices: []device.Device{
			{
				ID:    123,
				Brand: "brand",
				UUID:  "uuid",
				Type:  device.CPU,
			},
		},
	}

	envVars, err := getEnvVarsForLauncherManifest(ts,
		"masterHost", 8888, "certName", false, device.CUDA)

	assert.NilError(t, err)
	assert.Assert(t, len(envVars) > 0)

	assert.Equal(t, envVars["DET_MASTER"], "masterHost:8888")
	assert.Equal(t, envVars["DET_MASTER_HOST"], "masterHost")
	assert.Equal(t, envVars["DET_MASTER_IP"], "masterHost")
	assert.Equal(t, envVars["DET_MASTER_PORT"], "8888")
	assert.Equal(t, envVars["DET_CONTAINER_ID"], "Container_ID")
	assert.Equal(t, envVars["DET_CLUSTER_ID"], "Cluster_ID")
	assert.Equal(t, envVars["SLURM_KILL_BAD_EXIT"], "1")
	assert.Equal(t, envVars["DET_SLOT_TYPE"], "cuda")
	assert.Equal(t, envVars["DET_AGENT_ID"], "launcher")
	assert.Equal(t, envVars["DET_MASTER_CERT_NAME"], "certName")
	assert.Equal(t, envVars["DET_CONTAINER_LOCAL_TMP"], "1")
	assert.Equal(t, envVars["SINGULARITY_DOCKER_USERNAME"], "user")
	assert.Equal(t, envVars["APPTAINER_DOCKER_USERNAME"], "user")
	assert.Equal(t, envVars["SINGULARITY_DOCKER_PASSWORD"], "pwd")
	assert.Equal(t, envVars["APPTAINER_DOCKER_PASSWORD"], "pwd")

	assert.Equal(t, envVars["cpu"], "default")
	assert.Equal(t, envVars["myenv"], "xyz")
}

func Test_getEnvVarsForLauncherManifestErr(t *testing.T) {
	disableImageCache := true
	environment := expconf.EnvironmentConfigV0{
		RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
			RawCPU: []string{"cpudefault", "cpuexpconf"},
		},
		RawImage:            &expconf.EnvironmentImageMapV0{},
		RawRegistryAuth:     &types.AuthConfig{},
		RawForcePullImage:   &disableImageCache,
		RawPodSpec:          &expconf.PodSpec{},
		RawAddCapabilities:  []string{},
		RawDropCapabilities: []string{},
		RawPorts:            map[string]int{},
	}

	ts := &TaskSpec{
		Environment: environment,
	}

	_, err := getEnvVarsForLauncherManifest(ts, "masterHost", 8888, "certName", false, device.CUDA)
	assert.ErrorContains(t, err, "invalid user-defined environment variable 'cpudefault'")
}

func Test_generateRunDeterminedLinkNames(t *testing.T) {
	arg := []cproto.RunArchive{
		{
			Archive: archive.Archive{
				archive.Item{
					Path: "/determined_local_fs/dispatcher-wrapper.sh",
				},
				archive.Item{
					Path: "/" + etc.TaskLoggingSetupScriptResource,
				},
				archive.Item{
					Path: "/run/determined/link-1",
				},
				archive.Item{
					Path: "/run/determined/link-2",
				},
			},
		},
		{
			Archive: archive.Archive{
				archive.Item{
					Path: "/run/determined",
				},
				archive.Item{
					Path: "/run/determined/link-3",
				},
				archive.Item{
					Path: "/run/determined/xyz/abc.txt",
				},
			},
		},
	}

	want := []string{"link-1", "link-2", "link-3", "xyz"}

	got := generateRunDeterminedLinkNames(arg)

	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("generatedRunDeterminedLinkName = %v, want %v", got, want)
	}
}

func Test_getDataVolumns(t *testing.T) {
	arg := []mount.Mount{
		{
			Source: "xxx",
			Target: "yyy",
		},
		{
			Source: "aaa",
			Target: "bbb",
		},
		{
			Source: "/src",
			Target: "/tmp",
		},
	}

	wantName := []string{
		"ds0", "ds1", "ds2",
	}

	wantSource := []string{
		"xxx", "aaa", "/src",
	}

	wantTarget := []string{
		"yyy", "bbb", "/tmp",
	}

	volumns, mountOnTmp := getDataVolumes(arg)

	assert.Equal(t, mountOnTmp, true)
	for i, v := range volumns {
		assert.Equal(t, *v.Name, wantName[i])
		assert.Equal(t, *v.Source, wantSource[i])
		assert.Equal(t, *v.Target, wantTarget[i])
	}
}

func Test_getPayloadName(t *testing.T) {
	tests := []struct {
		name string
		desc string
		want string
	}{
		{
			name: "Test 1",
			desc: "abc_#123-&",
			want: "ai_abc_123-",
		},
		{
			name: "Test 2",
			desc: "   zyx-123 ",
			want: "ai_zyx-123",
		},
		{
			name: "Test 3",
			desc: "#sky , limit: .",
			want: "ai_skylimit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TaskSpec{}
			tr.Description = tt.desc
			got := getPayloadName(tr)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestTaskSpec_slotsPerNode(t *testing.T) {
	type fields struct {
		SlurmConfig expconf.SlurmConfig
		PbsConfig   expconf.PbsConfig
	}
	type args struct {
		isPbsLauncher bool
	}
	pbsSlotsPerNode := pbsSlotsPerNode
	slurmSlotsPerNode := slurmSlotsPerNode
	pbsConfig := expconf.PbsConfig{
		RawSlotsPerNode: &pbsSlotsPerNode,
		RawSbatchArgs:   []string{},
	}
	slurmConfig := expconf.SlurmConfig{
		RawSlotsPerNode: &slurmSlotsPerNode,
		RawGpuType:      nil,
		RawSbatchArgs:   []string{},
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name:   "Slots not specified, then get unspecified result, PBS case",
			fields: fields{},
			args:   args{isPbsLauncher: true},
			want:   unspecifiedSlotsPerNode,
		},
		{
			name:   "Slots not specified, then get unspecified result, Slurm case",
			fields: fields{},
			args:   args{isPbsLauncher: false},
			want:   unspecifiedSlotsPerNode,
		},
		{
			name: "Slots specified, then get specified value, PBS case",
			fields: fields{
				PbsConfig:   pbsConfig,
				SlurmConfig: slurmConfig,
			},
			args: args{isPbsLauncher: true},
			want: pbsSlotsPerNode,
		},
		{
			name: "Slots specified, then get specified value, Slurm case",
			fields: fields{
				PbsConfig:   pbsConfig,
				SlurmConfig: slurmConfig,
			},
			args: args{isPbsLauncher: false},
			want: slurmSlotsPerNode,
		},
		{
			name: "Slots specified for Slurm, but on PBS",
			fields: fields{
				SlurmConfig: slurmConfig,
			},
			args: args{isPbsLauncher: true},
			want: unspecifiedSlotsPerNode,
		},
		{
			name: "Slots specified for PBS, but on Slurm",
			fields: fields{
				PbsConfig: pbsConfig,
			},
			args: args{isPbsLauncher: false},
			want: unspecifiedSlotsPerNode,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TaskSpec{
				SlurmConfig: tt.fields.SlurmConfig,
				PbsConfig:   tt.fields.PbsConfig,
			}
			if got := tr.slotsPerNode(tt.args.isPbsLauncher); got != tt.want {
				t.Errorf("TaskSpec.slotsPerNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskSpec_computeResources(t *testing.T) {
	type fields struct {
		SlurmConfig expconf.SlurmConfig
		PbsConfig   expconf.PbsConfig
	}
	type args struct {
		tresSupported bool
		numSlots      int
		slotType      device.Type
		gresSupported bool
		isPbsLauncher bool
	}
	// Test data -- use different values for Slurm & PBS to detect any 'cross-over' errors
	slurmSlots := 32
	pbsSlots := 16
	slurmConfig := expconf.SlurmConfig{
		RawSlotsPerNode: &slurmSlots,
		RawGpuType:      nil,
		RawSbatchArgs:   []string{},
	}
	slurmConfigSlotsUnspecified := expconf.SlurmConfig{
		RawSlotsPerNode: nil,
		RawGpuType:      nil,
		RawSbatchArgs:   []string{},
	}
	pbsConfig := expconf.PbsConfig{
		RawSlotsPerNode: &pbsSlots,
		RawSbatchArgs:   []string{},
	}
	pbsConfigSlotsUnspecified := expconf.PbsConfig{
		RawSlotsPerNode: nil,
		RawSbatchArgs:   []string{},
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantResources *launcher.ResourceRequirements
		wantOpts      []string
	}{
		{
			name: "Slot type is CPU, Slurm, slots-per-node",
			fields: fields{
				SlurmConfig: slurmConfig,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CPU,
				gresSupported: false,
				isPbsLauncher: false,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": 4},
				Cores:     &map[string]float32{"per-node": float32(slurmSlots)},
			},
			wantOpts: []string{"--cpus-per-task=32"},
		},
		{
			name: "Slot type is CPU, PBS, slots-per-node",
			fields: fields{
				PbsConfig: pbsConfig,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CPU,
				gresSupported: false,
				isPbsLauncher: true,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": 7},
				Cores:     &map[string]float32{"per-node": float32(pbsSlots)},
			},
		},
		{
			name: "Slot type is CPU, Slurm, no slots_per_node",
			fields: fields{
				SlurmConfig: slurmConfigSlotsUnspecified,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CPU,
				gresSupported: false,
				isPbsLauncher: false,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": 100},
				Cores:     &map[string]float32{"per-node": 1},
			},
		},
		{
			name: "Slot type is CPU, PBS, no slots_per_node",
			fields: fields{
				PbsConfig: pbsConfigSlotsUnspecified,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CPU,
				gresSupported: false,
				isPbsLauncher: true,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": 100},
				Cores:     &map[string]float32{"per-node": 1},
			},
		},
		{
			name: "Slot type GPU, gres & tres supported (Slurm)",
			fields: fields{
				SlurmConfig: slurmConfig,
			},
			args: args{
				tresSupported: true,
				numSlots:      100,
				slotType:      device.CUDA,
				gresSupported: true,
				isPbsLauncher: false,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"per-node": 1},
				Gpus:      &map[string]int32{"total": int32(100)},
			},
			wantOpts: []string{"--gpus-per-task=32"},
		},
		{
			name: "Slot type GPU, gres & tres supported (Slurm), no slots_per_node",
			fields: fields{
				SlurmConfig: slurmConfigSlotsUnspecified,
			},
			args: args{
				tresSupported: true,
				numSlots:      100,
				slotType:      device.CUDA,
				gresSupported: true,
				isPbsLauncher: false,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"per-node": 1},
				Gpus:      &map[string]int32{"total": int32(100)},
			},
		},
		{
			name: "Slot type GPU, gres supported, PBS, no slots_per_node",
			fields: fields{
				PbsConfig: pbsConfigSlotsUnspecified,
			},
			args: args{
				tresSupported: true,
				numSlots:      100,
				slotType:      device.CUDA,
				gresSupported: true,
				isPbsLauncher: true,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"per-node": 1},
				Gpus:      &map[string]int32{"total": int32(100)},
			},
		},
		{
			name: "Slot type GPU, gres supported, PBS, slots-per-node",
			fields: fields{
				PbsConfig: pbsConfig,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CUDA,
				gresSupported: true,
				isPbsLauncher: true,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": 7},
				Gpus:      &map[string]int32{"per-node": int32(pbsSlots)},
			},
		},
		{
			name: "Slot type GPU, gres but not tres supported, Slurm",
			fields: fields{
				SlurmConfig: slurmConfig,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CUDA,
				gresSupported: true,
				isPbsLauncher: false,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": 4},
				Gpus:      &map[string]int32{"per-node": int32(32)},
			},
		},
		{
			name: "Slot type GPU, neither gres nor tres supported, Slurm, slots-per-node",
			fields: fields{
				SlurmConfig: slurmConfig,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CUDA,
				gresSupported: false,
				isPbsLauncher: false,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": 4},
			},
		},
		{
			name: "Slot type GPU, neither gres nor tres supported, PBS, slots-per-node",
			fields: fields{
				PbsConfig: pbsConfig,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CUDA,
				gresSupported: false,
				isPbsLauncher: true,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": 7},
			},
		},
		{
			name: "Slot type GPU, neither gres nor tres supported, Slurm, no slots-per-node",
			fields: fields{
				SlurmConfig: slurmConfigSlotsUnspecified,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CUDA,
				gresSupported: false,
				isPbsLauncher: false,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": int32(100)},
			},
		},
		{
			name: "Slot type GPU, neither gres nor tres supported, PBS, no slots-per-node",
			fields: fields{
				PbsConfig: pbsConfigSlotsUnspecified,
			},
			args: args{
				tresSupported: false,
				numSlots:      100,
				slotType:      device.CUDA,
				gresSupported: false,
				isPbsLauncher: true,
			},
			wantResources: &launcher.ResourceRequirements{
				Instances: &map[string]int32{"nodes": int32(100)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TaskSpec{
				SlurmConfig: tt.fields.SlurmConfig,
				PbsConfig:   tt.fields.PbsConfig,
			}
			got, gotOpts := tr.computeResources(tt.args.tresSupported, tt.args.numSlots, tt.args.slotType,
				tt.args.gresSupported, tt.args.isPbsLauncher)
			if !reflect.DeepEqual(got, tt.wantResources) {
				t.Errorf("TaskSpec.computeResources() = %v, want %v", got, tt.wantResources)
			}
			if !reflect.DeepEqual(gotOpts, tt.wantOpts) {
				t.Errorf("TaskSpec.computeResources() opts = %v, want %v", gotOpts, tt.wantOpts)
			}
		})
	}
}
