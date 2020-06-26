package tasks

import (
	"archive/tar"
	"fmt"
	"path/filepath"
	"strings"

	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	// ContainerWorkDir is working directory for containers.
	ContainerWorkDir  = "/run/determined/workdir"
	userPythonBaseDir = "/run/determined/pythonuserbase"
	runDir            = "/run/determined"
	rootDir           = "/"
)

func defaultEnvVars() map[string]string {
	// PYTHONUSERBASE allows us to `pip install --user` into a location guaranteed to be owned by
	// the user inside the container.
	envVars := map[string]string{"PYTHONUSERBASE": userPythonBaseDir}
	return envVars
}

// workDirArchive ensures that the workdir is created and owned by the user.
func workDirArchive(aug *model.AgentUserGroup) container.RunArchive {
	return wrapArchive(
		archive.Archive{
			aug.OwnedArchiveItem(runDir, nil, 0700, tar.TypeDir),
			aug.OwnedArchiveItem(ContainerWorkDir, nil, 0700, tar.TypeDir),
			aug.OwnedArchiveItem(userPythonBaseDir, nil, 0700, tar.TypeDir),
		},
		rootDir,
	)
}

// ToContainerSpec returns the container spec for associated task spec. This is a bridge method
// for the agent refactor project.
func ToContainerSpec(t TaskSpec) container.Spec {
	switch {
	case t.StartCommand != nil:
		return startCommand(t)
	case t.StartContainer != nil:
		return startContainer(t)
	case t.GCCheckpoints != nil:
		return gcCheckpoint(t)
	default:
		panic("unexpected task spec received")
	}
}

func getUser(agentUserGroup *model.AgentUserGroup) string {
	user := ""
	if agentUserGroup != nil {
		user = fmt.Sprintf("%d:%d", agentUserGroup.UID, agentUserGroup.GID)
	}
	return user
}

// ConfigureCommandEnvVars configures environment variables for cmd tasks.
func ConfigureCommandEnvVars(t TaskSpec) map[string]string {
	envVarsMap := defaultEnvVars()
	envVarsMap["DET_TASK_ID"] = t.TaskID
	return envVarsMap
}

// ConfigureCommandArchives returns the additional files for a c as an archive.
func ConfigureCommandArchives(t TaskSpec) []container.RunArchive {
	cmd := *t.StartCommand

	return []container.RunArchive{
		workDirArchive(cmd.AgentUserGroup),
		wrapArchive(cmd.AgentUserGroup.OwnArchive(cmd.UserFiles), ContainerWorkDir),
		wrapArchive(cmd.AdditionalFiles, rootDir),
		harnessArchive(t.HarnessPath, cmd.AgentUserGroup),
	}
}

func startCommand(t TaskSpec) container.Spec {
	cmd := *t.StartCommand
	user := getUser(cmd.AgentUserGroup)
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVarsMap := ConfigureCommandEnvVars(t)
	envVars := make([]string, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, fmt.Sprintf("%s=%s", envVarKey, envVarValue))
	}
	envVars = append(envVars, cmd.Config.Environment.EnvironmentVariables.For(deviceType)...)
	return container.Spec{
		PullSpec: container.PullSpec{
			Registry:  cmd.Config.Environment.RegistryAuth,
			ForcePull: cmd.Config.Environment.ForcePullImage,
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				User:         user,
				ExposedPorts: toPortSet(cmd.Config.Environment.Ports),
				Env:          envVars,
				Cmd:          cmd.Config.Entrypoint,
				Image:        cmd.Config.Environment.Image.For(deviceType),
				WorkingDir:   ContainerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     t.TaskContainerDefaults.NetworkMode,
				Mounts:          ToDockerMounts(cmd.Config.BindMounts),
				PublishAllPorts: true,
				ShmSize:         t.TaskContainerDefaults.ShmSizeBytes,
			},
			Archives: ConfigureCommandArchives(t),
		},
	}
}

// ConfigureTrialDockerMounts returns the host mounts for a trial container.
func ConfigureTrialDockerMounts(exp StartContainer) []mount.Mount {
	mounts := ToDockerMounts(exp.ExperimentConfig.BindMounts)
	if exp.ExperimentConfig.CheckpointStorage.SharedFSConfig != nil {
		sharedFS := exp.ExperimentConfig.CheckpointStorage.SharedFSConfig
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: sharedFS.HostPath,
			Target: model.DefaultSharedFSContainerPath,
			BindOptions: &mount.BindOptions{
				Propagation: model.DefaultSharedFSPropagation,
			},
		})
	}

	if exp.ExperimentConfig.DataLayer.SharedFSConfig != nil {
		SharedFSConfig := exp.ExperimentConfig.DataLayer.SharedFSConfig
		if SharedFSConfig.HostStoragePath != nil {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: *SharedFSConfig.HostStoragePath,
				Target: *SharedFSConfig.ContainerStoragePath,
			})
		}
	}
	if exp.ExperimentConfig.DataLayer.S3Config != nil {
		S3Config := exp.ExperimentConfig.DataLayer.S3Config
		if S3Config.LocalCacheHostPath != nil {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: *S3Config.LocalCacheHostPath,
				Target: *S3Config.LocalCacheContainerPath,
			})
		}
	}
	if exp.ExperimentConfig.DataLayer.GCSConfig != nil {
		GCSConfig := exp.ExperimentConfig.DataLayer.GCSConfig
		if GCSConfig.LocalCacheHostPath != nil {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: *GCSConfig.LocalCacheHostPath,
				Target: *GCSConfig.LocalCacheContainerPath,
			})
		}
	}

	return mounts
}

// ConfigureTrialEnvVars returns environment variables for a trial.
func ConfigureTrialEnvVars(t TaskSpec, rendezvousPorts []string) map[string]string {
	exp := *t.StartContainer

	networkInterface := t.TaskContainerDefaults.DtrainNetworkInterface
	if networkInterface == "" {
		networkInterface = "DET_AUTO_DETECT_NETWORK_INTERFACE"
	}

	envVars := defaultEnvVars()
	envVars["DET_EXPERIMENT_ID"] = fmt.Sprintf("%d", exp.InitialWorkload.ExperimentID)
	envVars["DET_TRIAL_ID"] = fmt.Sprintf("%d", exp.InitialWorkload.TrialID)
	envVars["DET_TRIAL_SEED"] = fmt.Sprintf("%d", exp.TrialSeed)
	envVars["DET_EXPERIMENT_CONFIG"] = jsonify(exp.ExperimentConfig)
	envVars["DET_HPARAMS"] = jsonify(exp.HParams)
	envVars["DET_INITIAL_WORKLOAD"] = jsonify(exp.InitialWorkload)
	envVars["DET_LATEST_CHECKPOINT"] = jsonify(exp.LatestCheckpoint)
	envVars["DET_WORKLOAD_MANAGER_TYPE"] = string(exp.WorkloadManagerType)
	envVars["DET_RENDEZVOUS_PORTS"] = strings.Join(rendezvousPorts, ",")
	envVars["DET_TRIAL_RUNNER_NETWORK_INTERFACE"] = networkInterface

	if t.TaskContainerDefaults.NCCLPortRange != "" {
		envVars["NCCL_PORT_RANGE"] = t.TaskContainerDefaults.NCCLPortRange
	}
	if t.TaskContainerDefaults.NCCLPortRange != "" {
		envVars["GLOO_PORT_RANGE"] = t.TaskContainerDefaults.NCCLPortRange
	}

	return envVars
}

// ConfigureTrialArchives returns the additional files for a trial as an archive.
func ConfigureTrialArchives(t TaskSpec) []container.RunArchive {
	exp := *t.StartContainer

	return []container.RunArchive{
		workDirArchive(exp.AgentUserGroup),
		wrapArchive(exp.AdditionalFiles, rootDir),
		wrapArchive(exp.AgentUserGroup.OwnArchive(exp.ModelDefinition), ContainerWorkDir),
		harnessArchive(t.HarnessPath, exp.AgentUserGroup),
	}
}

func startContainer(t TaskSpec) container.Spec {
	exp := *t.StartContainer
	user := getUser(exp.AgentUserGroup)
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	mounts := ConfigureTrialDockerMounts(exp)
	networkMode := t.TaskContainerDefaults.NetworkMode
	if exp.ExperimentConfig.Resources.SlotsPerTrial > 1 {
		networkMode = hostMode
	}
	rPorts := rendezvousPorts(t.Devices, networkMode)
	ports := make(nat.PortSet)
	var rPortsEnvVars []string
	for _, port := range rPorts {
		rPortsEnvVars = append(rPortsEnvVars, port.Port())
		ports[port] = struct{}{}
	}

	envVarsMap := ConfigureTrialEnvVars(t, rPortsEnvVars)
	envVars := make([]string, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, fmt.Sprintf("%s=%s", envVarKey, envVarValue))
	}
	envVars = append(envVars, exp.ExperimentConfig.Environment.EnvironmentVariables.For(deviceType)...)

	spec := container.Spec{
		PullSpec: container.PullSpec{
			ForcePull: exp.ExperimentConfig.Environment.ForcePullImage,
			Registry:  exp.ExperimentConfig.Environment.RegistryAuth,
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				Cmd:          []string{"/run/determined/workdir/entrypoint.sh"},
				User:         user,
				Image:        exp.ExperimentConfig.Environment.Image.For(deviceType),
				ExposedPorts: ports,
				Env:          envVars,
				WorkingDir:   ContainerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     networkMode,
				Mounts:          mounts,
				PublishAllPorts: true,
			},
			Archives: ConfigureTrialArchives(t),
		},
	}
	spec.RunSpec.HostConfig.ShmSize = t.TaskContainerDefaults.ShmSizeBytes
	if exp.ExperimentConfig.Resources.ShmSize != nil {
		spec.RunSpec.HostConfig.ShmSize = int64(*exp.ExperimentConfig.Resources.ShmSize)
	}
	return spec
}

// ConfigureGCEnvVars returns environment variables for gc.
func ConfigureGCEnvVars() map[string]string {
	return defaultEnvVars()
}

// ConfigureGCDockerMounts returns the host mounts for a gc container.
func ConfigureGCDockerMounts(gcc GCCheckpoints) []mount.Mount {
	mounts := ToDockerMounts(gcc.ExperimentConfig.BindMounts)
	if gcc.ExperimentConfig.CheckpointStorage.SharedFSConfig != nil {
		sharedFS := gcc.ExperimentConfig.CheckpointStorage.SharedFSConfig
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: sharedFS.HostPath,
			Target: model.DefaultSharedFSContainerPath,
			BindOptions: &mount.BindOptions{
				Propagation: model.DefaultSharedFSPropagation,
			},
		})
	}

	return mounts
}

// ConfigureGCArchives returns the additional files for gc as an archive.
func ConfigureGCArchives(t TaskSpec) []container.RunArchive {
	gcc := *t.GCCheckpoints

	return []container.RunArchive{
		workDirArchive(gcc.AgentUserGroup),
		wrapArchive(
			archive.Archive{
				gcc.AgentUserGroup.OwnedArchiveItem(
					"experiment_config.json",
					[]byte(jsonify(gcc.ExperimentConfig)),
					0600,
					tar.TypeReg,
				),
				gcc.AgentUserGroup.OwnedArchiveItem(
					"checkpoints_to_delete.json",
					[]byte(jsonify(gcc.ToDelete)),
					0600,
					tar.TypeReg,
				),
				gcc.AgentUserGroup.OwnedArchiveItem(
					etc.GCCheckpointsEntrypointResource,
					etc.MustStaticFile(etc.GCCheckpointsEntrypointResource),
					0700,
					tar.TypeReg,
				),
			},
			ContainerWorkDir,
		),
		harnessArchive(t.HarnessPath, gcc.AgentUserGroup),
	}
}

// ConfigureGCCmd configures the entrypoint for GC tasks.
func ConfigureGCCmd() []string {
	return []string{
		filepath.Join(ContainerWorkDir, etc.GCCheckpointsEntrypointResource),
		"--experiment-config",
		"experiment_config.json",
		"--delete",
		"checkpoints_to_delete.json",
	}
}

func gcCheckpoint(t TaskSpec) container.Spec {
	gcc := *t.GCCheckpoints
	user := getUser(gcc.AgentUserGroup)
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVarsMap := ConfigureGCEnvVars()
	envVars := make([]string, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, fmt.Sprintf("%s=%s", envVarKey, envVarValue))
	}
	envVars = append(envVars, gcc.ExperimentConfig.Environment.EnvironmentVariables.For(deviceType)...)

	return container.Spec{
		PullSpec: container.PullSpec{
			ForcePull: gcc.ExperimentConfig.Environment.ForcePullImage,
			Registry:  gcc.ExperimentConfig.Environment.RegistryAuth,
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				Cmd: []string{
					filepath.Join(ContainerWorkDir, etc.GCCheckpointsEntrypointResource),
					"--experiment-config",
					"experiment_config.json",
					"--delete",
					"checkpoints_to_delete.json",
				},
				User:       user,
				Image:      gcc.ExperimentConfig.Environment.Image.For(deviceType),
				Env:        envVars,
				WorkingDir: ContainerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     t.TaskContainerDefaults.NetworkMode,
				Mounts:          ConfigureGCDockerMounts(gcc),
				PublishAllPorts: true,
			},
			Archives: ConfigureGCArchives(t),
		},
	}
}
